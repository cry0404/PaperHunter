package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"PaperHunter/desktop/memory"
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

	mem, _ := memory.New("", 30, 7)
	if mem != nil {
		mem.Cleanup()
	}
	var recentKeys map[string]struct{}
	var recentEvents []memory.Event
	if mem != nil {
		if ks, err := mem.LoadRecentPaperKeys(); err == nil {
			recentKeys = ks
		} else {
			logger.Warn("读取记忆失败: %v", err)
		}
		if evs, err := mem.LoadEvents(7); err == nil {
			recentEvents = evs
		}
	}
	var profile *memory.ProfileCache
	if mem != nil && len(recentEvents) > 0 {
		embedFunc := func(texts []string) ([]float64, error) {
			// 暂不计算向量，返回 nil
			return nil, nil
		}
		profile = mem.BuildProfile(recentEvents, 12, embedFunc, "")
	}

	for _, seedPaper := range seeds {
		similarPapers, err := searchSimilarPapers(ctx, a, seedPaper, topK, fromDate, toDate)
		if err != nil {
			continue
		}

		filteredPapers := make([]*models.SimilarPaper, 0)
		for _, sp := range similarPapers {
			key := fmt.Sprintf("%s:%s", sp.Paper.Source, sp.Paper.SourceID)
			if recentKeys != nil {
				if _, exists := recentKeys[key]; exists {
					// 近期推送过的论文：降权但不直接过滤，保留丰富度
					sp.Similarity *= 0.7
				}
			}
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
			personalizedRerank(filteredPapers, profile)
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

	// 记录推荐事件到记忆，用于后续去重与画像
	if mem != nil && totalRecommended > 0 {
		var evs []memory.Event
		for _, group := range output.Recommendations {
			for _, sp := range group.Papers {
				evs = append(evs, memory.Event{
					Type:     "recommend_show",
					Source:   sp.Paper.Source,
					SourceID: sp.Paper.SourceID,
					Title:    sp.Paper.Title,
				})
			}
		}
		if err := mem.RecordRecommended(evs); err != nil {
			logger.Warn("记录记忆失败: %v", err)
		}
	}

	return string(data), nil
}

func personalizedRerank(papers []*models.SimilarPaper, profile *memory.ProfileCache) {
	if len(papers) <= 1 {
		return
	}
	sort.Slice(papers, func(i, j int) bool {
		return scorePaperWithProfile(papers[i], profile) > scorePaperWithProfile(papers[j], profile)
	})
}

// scorePaperWithProfile 计算混合得分：0.6*相似度 + 0.2*时间衰减 + 0.2*个性化
func scorePaperWithProfile(sp *models.SimilarPaper, profile *memory.ProfileCache) float64 {
	if sp == nil {
		return -1
	}
	sim := float64(sp.Similarity)
	recency := recencyScore(sp.Paper)
	personal := personalizationScore(sp, profile)
	return 0.6*sim + 0.2*recency + 0.2*personal
}

func recencyScore(p models.Paper) float64 {
	t := p.FirstAnnouncedAt
	if t.IsZero() {
		t = p.UpdatedAt
	}
	if t.IsZero() {
		return 0.5
	}
	days := time.Since(t).Hours() / 24
	halfLife := 60.0 // 约 2 个月衰减
	decay := math.Exp(-days / halfLife)
	if decay < 0 {
		return 0
	}
	return decay
}

func personalizationScore(sp *models.SimilarPaper, profile *memory.ProfileCache) float64 {
	if profile == nil {
		return 0
	}
	kwScore := keywordOverlapScore(sp.Paper.Title, profile.TopKeywords)
	platformScore := 0.0
	if profile.PlatformPreference != nil {
		if v, ok := profile.PlatformPreference[sp.Paper.Source]; ok {
			platformScore = v
		}
	}
	// 简单加权
	return 0.6*kwScore + 0.4*platformScore
}

func keywordOverlapScore(title string, topKeywords []string) float64 {
	if len(topKeywords) == 0 || title == "" {
		return 0
	}
	titleTokens := strings.Fields(strings.ToLower(title))
	set := make(map[string]struct{}, len(titleTokens))
	for _, t := range titleTokens {
		t = strings.Trim(t, " ,.;:()[]{}\"'`")
		if t != "" {
			set[t] = struct{}{}
		}
	}
	matches := 0
	for _, kw := range topKeywords {
		if _, ok := set[strings.ToLower(kw)]; ok {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return float64(matches) / float64(len(topKeywords))
}
