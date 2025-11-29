package ssrn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

	client := core.NewHTTPClient(int(config.Timeout.Seconds()), config.Proxy)
	return &Adapter{config: config, httpClient: client}, nil
}

func (a *Adapter) Name() string { return "ssrn" }

func (a *Adapter) GetConfig() platform.Config { return a.config }

//需要添加代理池等配置方案来为抓取提供效率，目前太慢了

func (a *Adapter) Search(ctx context.Context, q platform.Query) (platform.Result, error) {

	want := q.Limit
	if want <= 0 {
		want = a.config.PageSize * a.config.MaxPages
		if want <= 0 {
			want = 20
		}
	}

	startPage := q.Offset/a.config.PageSize + 1
	if startPage <= 0 {
		startPage = 1
	}

	ids := make([]string, 0, want)
	seen := map[string]struct{}{}
	maxPages := a.config.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}

	for p := 0; p < maxPages && len(ids) < want; p++ {
		npage := startPage + p
		searchURL := a.buildSearchURL(npage, q)
		logger.Debug("[SSRN] 搜索 URL(page=%d): %s", npage, searchURL)
		html, err := a.request(ctx, searchURL)
		if err != nil {
			if p == 0 {
				return platform.Result{}, fmt.Errorf("search request failed: %w", err)
			}
			logger.Warn("[SSRN] 第 %d 页请求失败: %v", npage, err)
			break
		}
		pageIDs := ExtractIDsFromSearchHTML(html)
		if len(pageIDs) == 0 {
			if p == 0 {
				logger.Info("[SSRN] 未找到论文条目")
			}
			break
		}
		for _, id := range pageIDs {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
			if len(ids) >= want {
				break
			}
		}
	}

	if len(ids) == 0 {
		return platform.Result{Total: 0, Papers: nil}, nil
	}

	target := len(ids)
	if q.Limit > 0 && q.Limit < target {
		target = q.Limit
	}

	papers := make([]*models.Paper, 0, target)
	delay := time.Duration(float64(time.Second) / a.config.RateLimitPerSecond)
	if delay < 300*time.Millisecond {
		delay = 300 * time.Millisecond
	}

	for i, id := range ids[:target] {
		select {
		case <-ctx.Done():
			return platform.Result{}, ctx.Err()
		default:
		}
		detailURL := a.config.BaseURL + "/sol3/papers.cfm?abstract_id=" + id
		logger.Debug("[SSRN] 抓取详情 %d/%d: %s", i+1, target, detailURL)
		dhtml, err := a.request(ctx, detailURL)
		if err != nil {
			logger.Warn("[SSRN] 详情抓取失败 id=%s: %v", id, err)
			continue
		}
		title, abs := ParseDetailTitleAbstract(dhtml)
		canonical, pdf := ParseDetailLinks(dhtml)
		if title == "" && abs == "" {
			logger.Warn("[SSRN] 解析失败 id=%s", id)
			continue
		}
		p := &models.Paper{
			Source:   "ssrn",
			SourceID: id,
			URL: func() string {
				if canonical != "" {
					return canonical
				}
				return detailURL
			}(),
			Title:    title,
			Abstract: abs,
		}
		if pdf != "" {
			if p.Comments == "" {
				p.Comments = "PDF: " + pdf
			} else {
				p.Comments += " | PDF: " + pdf
			}
		}
		papers = append(papers, p)
		time.Sleep(delay)
	}

	return platform.Result{Total: len(papers), Papers: papers}, nil
}

func (a *Adapter) buildSearchURL(npage int, q platform.Query) string {
	params := url.Values{}
	if len(q.Keywords) > 0 {

		params.Set("txtKey_Words", joinNonEmpty(q.Keywords, " "))
	}
	if npage <= 0 {
		npage = 1
	}
	params.Set("npage", fmt.Sprintf("%d", npage))
	params.Set("sort", a.config.Sort)
	params.Set("stype", "abs")
	if a.config.PageSize > 0 {
		params.Set("lim", fmt.Sprintf("%d", a.config.PageSize))
	}
	return a.config.BaseURL + "/sol3/results.cfm?" + params.Encode()
}

func (a *Adapter) request(ctx context.Context, u string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			// 指数退避：1s, 2s, 4s, 8s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests { // 429
			lastErr = fmt.Errorf("HTTP error: 429")
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", lastErr
}

func joinNonEmpty(ss []string, sep string) string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, sep)
}
