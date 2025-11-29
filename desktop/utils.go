package main

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"
)

// getDailyRecommendationsDirect 直接调用工具逻辑（回退方案）
func (a *App) getDailyRecommendationsDirect(opts RecommendOptions, agentLogs []AgentLogEntry, intent *UserIntent) (string, error) {
	topK := opts.TopK
	if topK <= 0 {
		topK = 5
	}
	maxRecommendations := opts.MaxRecommendations
	if maxRecommendations <= 0 {
		maxRecommendations = 20
	}

	ctx := context.Background()

	// 直接调用推荐逻辑（复用 zoteroRecommendTool 中的逻辑）
	output := &ZoteroRecommendOutput{

		Recommendations: make([]RecommendationGroup, 0),
	}

	// 检查今天是否已爬取
	today := time.Now().Format("2006-01-02")
	alreadyCrawled := checkTodayCrawled()
	output.CrawledToday = alreadyCrawled

	// 日期范围（用于后续搜索）
	dateFrom := opts.DateFrom
	if dateFrom == "" {
		dateFrom = today
	}
	dateTo := opts.DateTo
	if dateTo == "" {
		dateTo = today
	}

	// 如果需要爬取（未爬取或强制爬取）
	if !alreadyCrawled || opts.ForceCrawl {
		logger.Info("使用 New Submissions 页面爬取今日 arXiv CS 论文...")

		// 使用新的 New Submissions 爬取方式
		crawlCount, err := crawlTodayNewSubmissions(ctx, a, "cs")
		if err != nil {
			logger.Warn("爬取失败: %v", err)
			// 爬取失败不影响继续执行
		} else {
			output.ArxivCrawlCount = crawlCount
			// 如果有论文被爬取，标记今天已爬取
			if crawlCount > 0 {
				if err := markTodayCrawled(); err == nil {
					output.CrawledToday = true
				}
			}
		}
	}

	// 收集推荐种子
	var seeds []*models.Paper

	zoteroPapers, err := getZoteroPapers(opts.ZoteroCollection, 50)
	if err != nil {
		// 记录错误但不中断，继续尝试其他来源
		logger.Warn("从 Zotero 获取论文失败: %v", err)
	} else {
		seeds = append(seeds, zoteroPapers...)
		// 稍后设置种子论文数量
	}

	// 2. 添加 Hype Layer 生成的示例论文
	if intent != nil && intent.GeneratedTitle != "" && intent.GeneratedAbstract != "" {
		seeds = append(seeds, &models.Paper{
			Title:    intent.GeneratedTitle,
			Abstract: intent.GeneratedAbstract,
			Source:   "user_query",
			SourceID: "hype_generated",
		})
	}

	if len(seeds) == 0 {
		// 记录警告到日志
		warnLog := AgentLogEntry{
			Type:      "error",
			Content:   "未找到种子论文（Zotero 为空且未生成示例）",
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, warnLog)
		a.logAndEmit(warnLog)

		recommendResult := RecommendResult{
			CrawledToday:    output.CrawledToday,
			ArxivCrawlCount: output.ArxivCrawlCount,
			SeedPaperCount:  len(seeds),
			Recommendations: make([]RecommendationGroup, 0),
			Message:         "未找到种子论文",
			AgentLogs:       agentLogs,
		}
		data, marshalErr := json.Marshal(recommendResult)
		if marshalErr != nil {
			return "", fmt.Errorf("marshal empty result failed: %w", marshalErr)
		}
		return string(data), nil
	}

	// 解析日期范围用于搜索（默认使用今天的日期）
	var fromDate, toDate *time.Time

	// 确定搜索日期范围（与爬取日期范围一致）
	searchDateFrom := dateFrom
	searchDateTo := dateTo

	from, err := time.Parse("2006-01-02", searchDateFrom)
	if err == nil {
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		fromDate = &from
	}
	to, err := time.Parse("2006-01-02", searchDateTo)
	if err == nil {
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
		toDate = &to
	}

	logger.Info("搜索日期范围: %s 至 %s", searchDateFrom, searchDateTo)

	// 为每篇种子论文搜索相似的新论文
	allRecommendedPapers := make(map[string]*models.SimilarPaper)

	for _, seedPaper := range seeds {
		similarPapers, err := searchSimilarPapers(ctx, a, seedPaper, topK, fromDate, toDate)
		if err != nil {
			continue
		}

		// 过滤重复
		filteredPapers := make([]*models.SimilarPaper, 0)
		for _, sp := range similarPapers {
			key := fmt.Sprintf("%s:%s", sp.Paper.Source, sp.Paper.SourceID)
			if _, exists := allRecommendedPapers[key]; !exists {
				isDuplicate := false
				for _, s := range seeds {
					if s.Source == sp.Paper.Source && s.SourceID == sp.Paper.SourceID {
						isDuplicate = true
						break
					}
				}
				if !isDuplicate {
					filteredPapers = append(filteredPapers, sp)
					allRecommendedPapers[key] = sp
				}
			}
		}

		if len(filteredPapers) > 0 {
			output.Recommendations = append(output.Recommendations, RecommendationGroup{
				SeedPaper: *seedPaper,
				Papers:    filteredPapers,
			})
		}

		if len(allRecommendedPapers) >= maxRecommendations {
			break
		}
	}

	// 限制总推荐数量
	if len(allRecommendedPapers) > maxRecommendations {
		total := 0
		for i := range output.Recommendations {
			if total >= maxRecommendations {
				output.Recommendations = output.Recommendations[:i]
				break
			}
			total += len(output.Recommendations[i].Papers)
		}
	}

	totalRecommended := 0
	for _, group := range output.Recommendations {
		totalRecommended += len(group.Papers)
	}

	// 记录搜索结果到日志
	if totalRecommended == 0 {
		notFoundLog := AgentLogEntry{
			Type:      "assistant",
			Content:   fmt.Sprintf("未找到匹配的推荐论文。已搜索 %d 篇种子论文，但未找到相似的新论文。", len(seeds)),
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, notFoundLog)
		a.logAndEmit(notFoundLog)
		output.Message = fmt.Sprintf("未找到匹配的推荐论文，基于 %d 篇种子论文", len(seeds))
	} else {
		successLog := AgentLogEntry{
			Type:      "assistant",
			Content:   fmt.Sprintf("成功推荐 %d 篇论文，基于 %d 篇种子论文", totalRecommended, len(seeds)),
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, successLog)
		a.logAndEmit(successLog)
		output.Message = fmt.Sprintf("成功推荐 %d 篇论文，基于 %d 篇种子论文", totalRecommended, len(seeds))
	}

	// 确保日志不为空（至少包含用户查询）
	if len(agentLogs) == 0 {
		logger.Warn("直接调用模式下，日志为空，添加默认日志")
		userLog := AgentLogEntry{
			Type:      "user",
			Content:   "获取每日推荐",
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, userLog)
		a.logAndEmit(userLog)

		assistantLog := AgentLogEntry{
			Type:      "assistant",
			Content:   "使用直接调用方式获取推荐（未通过 agent）",
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, assistantLog)
		a.logAndEmit(assistantLog)
	}

	logger.Info("准备返回推荐结果，包含 %d 条日志", len(agentLogs))

	// 转换为前端格式
	recommendResult := RecommendResult{
		CrawledToday:    output.CrawledToday,
		ArxivCrawlCount: output.ArxivCrawlCount,
		SeedPaperCount:  len(seeds),
		Recommendations: output.Recommendations,
		Message:         output.Message,
		AgentLogs:       agentLogs,
	}

	data, err := json.Marshal(recommendResult)
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}

	logger.Debug("返回的 JSON 数据长度: %d 字节", len(data))
	logger.Debug("返回的 JSON 数据预览: %s", string(data)[:min(500, len(data))])

	return string(data), nil
}
