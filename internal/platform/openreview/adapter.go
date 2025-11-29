package openreview

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"PaperHunter/internal/core"
	"PaperHunter/internal/models"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"
)

type Adapter struct {
	config     *Config
	httpClient *http.Client
}

func NewAdapter(config *Config) (*Adapter, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	client := core.NewHTTPClient(config.Timeout, config.Proxy)
	return &Adapter{config: config, httpClient: client}, nil
}

func (a *Adapter) Name() string { return "openreview" }

func (a *Adapter) GetConfig() platform.Config { return a.config }

// Search 实现 Platform 接口
func (a *Adapter) Search(ctx context.Context, q platform.Query) (platform.Result, error) {
	// OpenReview 使用 venue_id 而非通用 categories
	// 这里需要特殊处理：从 Categories 提取 venue_id
	if len(q.Categories) == 0 {
		return platform.Result{}, fmt.Errorf("openreview requires venue_id in categories")
	}
	venueID := q.Categories[0] // 如 "ICLR.cc/2026/Conference/Submission"

	var allPapers []*models.Paper
	offset := q.Offset
	userLimit := q.Limit
	if userLimit == 0 {
		userLimit = 1000 // 默认最多获取 1000 篇
	}

	// 每次分页请求的数量（API 限制）
	pageSize := 100
	if userLimit < pageSize {
		pageSize = userLimit
	}

	for len(allPapers) < userLimit {

		remaining := userLimit - len(allPapers)
		currentLimit := pageSize
		if remaining < currentLimit {
			currentLimit = remaining
		}

		params := url.Values{}
		params.Add("content.venueid", venueID)
		params.Add("details", "replyCount,invitation")
		params.Add("limit", fmt.Sprintf("%d", currentLimit))
		params.Add("offset", fmt.Sprintf("%d", offset))
		params.Add("sort", "number:desc")

		apiURL := a.config.APIBase + "/notes?" + params.Encode()
		logger.Debug("[OpenReview] 请求 API: offset=%d, limit=%d", offset, currentLimit)
		body, err := a.request(ctx, apiURL)
		if err != nil {
			return platform.Result{}, err
		}

		resp, err := parseResponse(body)
		if err != nil {
			return platform.Result{}, err
		}

		if len(resp.Notes) == 0 {
			logger.Debug("[OpenReview] 无更多论文，停止分页")
			break
		}

		logger.Debug("[OpenReview] 本次获取 %d 篇论文", len(resp.Notes))
		allPapers = append(allPapers, resp.Notes...)
		offset += len(resp.Notes)

		// 如果返回数量少于请求数量，说明已无更多
		if len(resp.Notes) < currentLimit {
			logger.Debug("[OpenReview] 已到最后一页")
			break
		}

		// 已达到用户指定数量
		if len(allPapers) >= userLimit {
			logger.Debug("[OpenReview] 已达到用户指定数量 %d", userLimit)
			break
		}

		// 分页间隔，避免触发频率限制
		logger.Debug("[OpenReview] 等待 1s 后请求下一页...")
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return platform.Result{}, ctx.Err()
		}
	}

	// 截断到精确数量
	if len(allPapers) > userLimit {
		allPapers = allPapers[:userLimit]
	}

	return platform.Result{
		Total:  len(allPapers),
		Papers: allPapers,
	}, nil
}

func (a *Adapter) request(ctx context.Context, apiURL string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {

			waitTime := time.Duration(2<<uint(attempt-1)) * time.Second
			logger.Warn("[OpenReview] 重试第 %d 次，等待 %v...", attempt, waitTime)
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 4 {
				continue
			}
			break
		}
		defer resp.Body.Close()

		// 429 Too Many Requests - 重试
		if resp.StatusCode == 429 {
			logger.Debug("[OpenReview] 收到 429 频率限制，尝试=%d", attempt+1)
			lastErr = fmt.Errorf("rate limited (429)")
			if attempt < 4 {
				continue
			}
			logger.Error("[OpenReview] 超出重试次数，请稍后再试或配置代理")
			return "", fmt.Errorf("rate limit exceeded after %d attempts (wait ~1min and retry)", attempt+1)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	return "", lastErr
}
