package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"
)

// getDailyRecommendationsDirect
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

	output := &ZoteroRecommendOutput{

		Recommendations: make([]RecommendationGroup, 0),
	}

	today := time.Now().Format("2006-01-02")
	alreadyCrawled := checkTodayCrawled()
	output.CrawledToday = alreadyCrawled

	dateFrom := opts.DateFrom
	if dateFrom == "" {
		dateFrom = today
	}
	dateTo := opts.DateTo
	if dateTo == "" {
		dateTo = today
	}

	if !alreadyCrawled || opts.ForceCrawl {
		logger.Info("使用 New Submissions 页面爬取今日 arXiv CS 论文...")

		crawlCount, err := crawlTodayNewSubmissions(ctx, a, "cs")
		if err != nil {
			logger.Warn("爬取失败: %v", err)

		} else {
			output.ArxivCrawlCount = crawlCount

			if crawlCount > 0 {
				if err := markTodayCrawled(); err == nil {
					output.CrawledToday = true
				}
			}
		}
	}

	var seeds []*models.Paper

	zoteroPapers, err := getZoteroPapers(opts.ZoteroCollection, 50)
	if err != nil {

		logger.Warn("从 Zotero 获取论文失败: %v", err)
	} else {
		seeds = append(seeds, zoteroPapers...)

	}

	if intent != nil && intent.GeneratedTitle != "" && intent.GeneratedAbstract != "" {
		seeds = append(seeds, &models.Paper{
			Title:    intent.GeneratedTitle,
			Abstract: intent.GeneratedAbstract,
			Source:   "user_query",
			SourceID: "hype_generated",
		})
	}

	if len(seeds) == 0 {

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

	var fromDate, toDate *time.Time

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

	allRecommendedPapers := make(map[string]*models.SimilarPaper)

	for _, seedPaper := range seeds {
		similarPapers, err := searchSimilarPapers(ctx, a, seedPaper, topK, fromDate, toDate)
		if err != nil {
			continue
		}

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
			rerankByScore(filteredPapers)
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


func rerankByScore(papers []*models.SimilarPaper) {
	if len(papers) <= 1 {
		return
	}
	sort.Slice(papers, func(i, j int) bool {
		return scorePaper(papers[i]) > scorePaper(papers[j])
	})
}

// scorePaper 计算混合得分：0.7*相似度 + 0.3*时间衰减
func scorePaper(sp *models.SimilarPaper) float64 {
	if sp == nil {
		return -1
	}
	sim := float64(sp.Similarity)
	recency := recencyScore(sp.Paper)
	return 0.9*sim + 0.1*recency
}


func recencyScore(p models.Paper) float64 {
	t := p.FirstAnnouncedAt
	if t.IsZero() {
		t = p.UpdatedAt
	}
	if t.IsZero() {
		return 0.5 // 无时间信息时给予中性分
	}
	days := time.Since(t).Hours() / 24
	halfLife := 60.0 // 约 2 个月衰减
	decay := math.Exp(-days / halfLife)
	if decay < 0 {
		return 0
	}
	return decay
}
