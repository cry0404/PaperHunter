package arxiv

import (
	"context"
	"fmt"
	"io"
	"log"
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

	client := core.NewHTTPClient(config.Timeout, config.Proxy)

	return &Adapter{
		config:     config,
		httpClient: client,
	}, nil
}

func (a *Adapter) Name() string { return "arxiv" }

func (a *Adapter) GetConfig() platform.Config { return a.config }

func (a *Adapter) Search(ctx context.Context, q platform.Query) (platform.Result, error) {
	if a.config.UseAPI {
		return a.searchViaAPI(ctx, q)
	}
	return a.searchViaWeb(ctx, q)
}

// searchViaAPI 使用官方 API 搜索（支持分页）
func (a *Adapter) searchViaAPI(ctx context.Context, q platform.Query) (platform.Result, error) {
	searchQuery := a.buildAPIQuery(q)

	// 确定目标数量和每页大小
	targetLimit := q.Limit
	if targetLimit == 0 {
		targetLimit = 20000
	}
	pageSize := a.config.Step
	if pageSize > 200 {
		pageSize = 200 // arXiv API 单次最大 200, 我自己的测试
	}

	var allPapers []*models.Paper
	totalFound := 0
	start := q.Offset

	for {

		remaining := targetLimit - len(allPapers)
		if remaining <= 0 {
			break
		}
		currentPageSize := pageSize
		if remaining < pageSize {
			currentPageSize = remaining
		}

		params := url.Values{}
		params.Add("search_query", searchQuery)
		params.Add("start", fmt.Sprintf("%d", start))
		params.Add("max_results", fmt.Sprintf("%d", currentPageSize))
		params.Add("sortBy", "submittedDate")
		params.Add("sortOrder", "descending")

		apiURL := a.config.APIBase + "?" + params.Encode()
		logger.Debug("[arXiv] API 请求: start=%d, max=%d", start, currentPageSize)

		content, err := a.request(ctx, apiURL)
		if err != nil {
			return platform.Result{}, fmt.Errorf("API request failed: %w", err)
		}

		papers, total, err := ParseAtomFeed(content)
		if err != nil {
			return platform.Result{}, fmt.Errorf("failed to parse API response: %w", err)
		}

		if totalFound == 0 {
			totalFound = total
			logger.Info("[arXiv] 总共找到 %d 篇论文，开始分页抓取", total)
		}

		allPapers = append(allPapers, papers...)
		logger.Info("[arXiv] 已抓取 %d/%d 篇", len(allPapers), totalFound)

		// 判断是否结束
		if len(papers) == 0 || len(allPapers) >= totalFound {
			break
		}

		start += len(papers)
		time.Sleep(1000 * time.Millisecond) //防止触发 429
	}

	logger.Info("[arXiv] API 抓取完成，共 %d 篇论文", len(allPapers))
	return platform.Result{Total: totalFound, Papers: allPapers}, nil
}

// searchViaWeb 使用网页搜索（支持分页）
func (a *Adapter) searchViaWeb(ctx context.Context, q platform.Query) (platform.Result, error) {

	webURL := a.buildWebQuery(q)
	logger.Debug("[arXiv] Web 请求第 1 页")

	content, err := a.request(ctx, webURL)
	if err != nil {
		return platform.Result{}, fmt.Errorf("web request failed: %w", err)
	}

	papers, totalFound, err := ParseSearchHTML(content)
	if err != nil {
		return platform.Result{}, fmt.Errorf("failed to parse web response: %w", err)
	}

	logger.Info("[arXiv] 总共找到 %d 篇论文，第 1 页返回 %d 篇", totalFound, len(papers))

	targetLimit := q.Limit
	if targetLimit == 0 || targetLimit > totalFound {
		targetLimit = totalFound
	}

	pageSize := a.config.Step
	if pageSize < 50 {
		pageSize = 50 // arXiv web 最小 50
	}

	for len(papers) < targetLimit && len(papers) < totalFound {
		offset := len(papers)
		q.Offset = offset
		webURL := a.buildWebQuery(q)
		logger.Debug("[arXiv] Web 请求第 %d 页 (offset=%d)", offset/pageSize+1, offset)

		content, err := a.request(ctx, webURL)
		if err != nil {
			logger.Warn("[arXiv] 抓取第 %d 页失败: %v", offset/pageSize+1, err)
			break
		}

		pagePapers, _, err := ParseSearchHTML(content)
		if err != nil {
			logger.Warn("[arXiv] 解析第 %d 页失败: %v", offset/pageSize+1, err)
			break
		}

		if len(pagePapers) == 0 {
			break
		}

		papers = append(papers, pagePapers...)
		logger.Info("[arXiv] 已抓取 %d/%d 篇", len(papers), totalFound)

		time.Sleep(500 * time.Millisecond) // 限流保护
	}

	if q.Limit > 0 && len(papers) > q.Limit {
		logger.Debug("[arXiv] 截断结果从 %d 到 %d 篇", len(papers), q.Limit)
		papers = papers[:q.Limit]
	}

	logger.Info("[arXiv] Web 抓取完成，共 %d 篇论文", len(papers))
	return platform.Result{Total: totalFound, Papers: papers}, nil
}

// buildAPIQuery 构建 API 查询字符串
func (a *Adapter) buildAPIQuery(q platform.Query) string {
	var parts []string

	for _, kw := range q.Keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if strings.Contains(kw, " ") {
			kw = fmt.Sprintf(`"%s"`, kw)
		}
		parts = append(parts, fmt.Sprintf("all:%s", kw))
	}

	for _, cat := range q.Categories {
		cat = strings.TrimSpace(cat)
		if cat == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("cat:%s", cat))
	}

	if len(parts) == 0 {
		log.Printf("no keywords or categories provided, fallback to cat:cs.*")
		return "cat:cs.*"
	}

	query := strings.Join(parts, " AND ")

	if q.DateFrom != "" || q.DateTo != "" {
		from := "*"
		to := "*"
		if q.DateFrom != "" {
			if t, err := time.Parse("2006-01-02", q.DateFrom); err == nil {
				from = t.Format("200601021504")
			}
		}
		if q.DateTo != "" {
			if t, err := time.Parse("2006-01-02", q.DateTo); err == nil {
				to = t.Format("200601021504")
			}
		}
		query += fmt.Sprintf(" AND submittedDate:[%s TO %s]", from, to)
	}

	return query
}

func (a *Adapter) buildWebQuery(q platform.Query) string {
	params := url.Values{}
	params.Add("advanced", "1")

	termIndex := 0

	// 关键词：默认 OR 连接，仅在标题中搜索
	for _, kw := range q.Keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if strings.Contains(kw, " ") && !(strings.HasPrefix(kw, `"`) && strings.HasSuffix(kw, `"`)) {
			kw = fmt.Sprintf(`"%s"`, kw)
		}
		if termIndex > 0 {
			params.Add(fmt.Sprintf("terms-%d-operator", termIndex), "OR")
		}
		params.Add(fmt.Sprintf("terms-%d-term", termIndex), kw)
		params.Add(fmt.Sprintf("terms-%d-field", termIndex), "all") //这里 title 是标题、all是所有 abs 是摘要 ti 是标题
		termIndex++
	}

	// 类别：与关键词块 AND，内部 OR
	for i, cat := range q.Categories {
		cat = strings.TrimSpace(cat)
		if cat == "" {
			continue
		}
		operator := "AND"
		if i > 0 {
			operator = "OR"
		}
		if termIndex > 0 {
			params.Add(fmt.Sprintf("terms-%d-operator", termIndex), operator)
		}
		params.Add(fmt.Sprintf("terms-%d-term", termIndex), cat)
		params.Add(fmt.Sprintf("terms-%d-field", termIndex), "cross_list_category")
		termIndex++
	}

	params.Add("classification-include_cross_list", "include")
	params.Add("abstracts", "show")

	pageSize := q.Limit
	if pageSize == 0 || pageSize < 50 {
		pageSize = 50
	}
	params.Add("size", fmt.Sprintf("%d", pageSize))
	params.Add("order", "-announced_date_first")
	if q.Offset > 0 {
		params.Add("start", fmt.Sprintf("%d", q.Offset))
	}

	// 日期范围过滤 - 尝试多种格式
	if q.DateFrom != "" || q.DateTo != "" {
		params.Add("date-filter_by", "date_range")

		if q.DateFrom != "" {
			if dateFrom, err := time.Parse("2006-01-02", q.DateFrom); err == nil {
				// arXiv web 搜索需要 YYYY-MM-DD 格式
				params.Add("date-from_date", dateFrom.Format("2006-01-02"))
			}
		}

		if q.DateTo != "" {
			if dateTo, err := time.Parse("2006-01-02", q.DateTo); err == nil {
				// arXiv web 搜索需要 YYYY-MM-DD 格式
				params.Add("date-to_date", dateTo.Format("2006-01-02"))
			}
		}
	}

	webURL := a.config.WebBase + "?" + params.Encode()
	logger.Debug("[arXiv] 构建的 URL: %s", webURL)
	return webURL
}

func (a *Adapter) request(ctx context.Context, url string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		// 这里的 User-Agent 可以增加点随机，但经过实际测试发现似乎没有什么影响
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36")

		resp, err := a.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
				continue
			}
			break
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP error: %d", resp.StatusCode)
			if attempt < 2 {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
				continue
			}
			break
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}
		return string(body), nil
	}
	return "", lastErr
}
