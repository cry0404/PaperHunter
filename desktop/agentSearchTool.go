package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"PaperHunter/pkg/logger"
)

// AgentSearchTool 简化版的智能搜索工具
type AgentSearchTool struct {
	client    *http.Client
	cache     map[string]*CacheEntry

}

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      any
	ExpiresAt time.Time
}

// SearchContext 搜索上下文信息
type SearchContext struct {
	AvailableVenues    []VenueInfo    `json:"available_venues"`
	ArxivCategories    []CategoryInfo `json:"arxiv_categories"`
	TrendingKeywords   []string       `json:"trending_keywords"`
	CurrentSeason      string         `json:"current_season"`
	UpcomingDeadlines  []DeadlineInfo `json:"upcoming_deadlines"`
}

// VenueInfo 会议信息
type VenueInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Status      string   `json:"status"` // "active", "upcoming", "closed"
	Keywords    []string `json:"keywords"`
	Field       string   `json:"field"`
	Deadline    string   `json:"deadline,omitempty"`
}

// CategoryInfo arXiv 分类信息
type CategoryInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
	Related     []string `json:"related_categories"`
}

// DeadlineInfo 截止日期信息
type DeadlineInfo struct {
	VenueName string `json:"venue_name"`
	Deadline  string `json:"deadline"`
	Type      string `json:"type"` // "submission", "notification", "camera_ready"
}

// EnhancedSearchQuery 增强的搜索查询
type EnhancedSearchQuery struct {
	OriginalQuery        string            `json:"original_query"`
	OpenReviewVenue      string            `json:"openreview_venue"`
	ArxivQuery           string            `json:"arxiv_query"`
	RecommendedVenues    []string          `json:"recommended_venues"`
	RecommendedCategories []string         `json:"recommended_categories"`
	ExpandedKeywords     []string          `json:"expanded_keywords"`
	SearchStrategy       string            `json:"search_strategy"`
	Confidence           float64           `json:"confidence"`
	Context              *SearchContext    `json:"context,omitempty"`
}

// NewAgentSearchTool 创建 AgentSearchTool 实例
func NewAgentSearchTool() *AgentSearchTool {
	return &AgentSearchTool{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]*CacheEntry),
	}
}



// TODO：持续性缓存部分
func (ast *AgentSearchTool) GetSearchContext(ctx context.Context) (*SearchContext, error) {

	cacheKey := "search_context"
	if entry, exists := ast.cache[cacheKey]; exists && entry.ExpiresAt.After(time.Now()) {
		return entry.Data.(*SearchContext), nil
	}


	searchContext := &SearchContext{
		AvailableVenues:   ast.getStaticVenueInfo(),
		ArxivCategories:   ast.getStaticArxivCategories(),
		TrendingKeywords:  ast.getCurrentTrendingKeywords(),
		CurrentSeason:     ast.getCurrentSeason(),
	}

	// TODO： 将缓存结果导出成本地 json 文件
	ast.cache[cacheKey] = &CacheEntry{
		Data:      searchContext,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	logger.Info("AgentSearchTool: 已构建搜索上下文，包含 %d 个会议和 %d 个分类",
		len(searchContext.AvailableVenues), len(searchContext.ArxivCategories))

	return searchContext, nil
}


// TODO ：下面的静态信息都应该改为 agenticSearch 获取
// getStaticVenueInfo 获取静态会议信息（2024-2025年主要会议）
func (ast *AgentSearchTool) getStaticVenueInfo() []VenueInfo {
	return []VenueInfo{
		{
			ID:          "NeurIPS.cc/2024/Conference",
			Name:        "NeurIPS 2024",
			DisplayName: "Conference on Neural Information Processing Systems 2024",
			Status:      "active",
			Keywords:    []string{"neural networks", "machine learning", "deep learning", "AI", "optimization"},
			Field:       "Machine Learning",
		},
		{
			ID:          "NeurIPS.cc/2025/Conference",
			Name:        "NeurIPS 2025",
			DisplayName: "Conference on Neural Information Processing Systems 2025",
			Status:      "upcoming",
			Keywords:    []string{"neural networks", "machine learning", "deep learning", "AI", "optimization"},
			Field:       "Machine Learning",
			Deadline:    "2025-05-27",
		},
		{
			ID:          "ICLR.cc/2025/Conference",
			Name:        "ICLR 2025",
			DisplayName: "International Conference on Learning Representations 2025",
			Status:      "upcoming",
			Keywords:    []string{"deep learning", "representation learning", "neural networks", "transformers"},
			Field:       "Deep Learning",
			Deadline:    "2025-09-25",
		},
		{
			ID:          "ICML.cc/2024/Conference",
			Name:        "ICML 2024",
			DisplayName: "International Conference on Machine Learning 2024",
			Status:      "active",
			Keywords:    []string{"machine learning", "algorithms", "theory", "optimization"},
			Field:       "Machine Learning",
		},
		{
			ID:          "ICML.cc/2025/Conference",
			Name:        "ICML 2025",
			DisplayName: "International Conference on Machine Learning 2025",
			Status:      "upcoming",
			Keywords:    []string{"machine learning", "algorithms", "theory", "optimization"},
			Field:       "Machine Learning",
			Deadline:    "2025-01-30",
		},
		{
			ID:          "ACL/2024/Conference",
			Name:        "ACL 2024",
			DisplayName: "Annual Meeting of the Association for Computational Linguistics 2024",
			Status:      "active",
			Keywords:    []string{"NLP", "computational linguistics", "language models", "transformers"},
			Field:       "Natural Language Processing",
		},
		{
			ID:          "ACL/2025/Conference",
			Name:        "ACL 2025",
			DisplayName: "Annual Meeting of the Association for Computational Linguistics 2025",
			Status:      "upcoming",
			Keywords:    []string{"NLP", "computational linguistics", "language models", "transformers"},
			Field:       "Natural Language Processing",
			Deadline:    "2025-02-24",
		},
		{
			ID:          "EMNLP/2024/Conference",
			Name:        "EMNLP 2024",
			DisplayName: "Empirical Methods in Natural Language Processing 2024",
			Status:      "active",
			Keywords:    []string{"NLP", "empirical methods", "language understanding", "text processing"},
			Field:       "Natural Language Processing",
		},
		{
			ID:          "AAAI/2025/Conference",
			Name:        "AAAI 2025",
			DisplayName: "AAAI Conference on Artificial Intelligence 2025",
			Status:      "upcoming",
			Keywords:    []string{"artificial intelligence", "knowledge representation", "reasoning", "planning"},
			Field:       "Artificial Intelligence",
			Deadline:    "2024-08-08",
		},
		{
			ID:          "IJCAI/2025/Conference",
			Name:        "IJCAI 2025",
			DisplayName: "International Joint Conference on Artificial Intelligence 2025",
			Status:      "upcoming",
			Keywords:    []string{"artificial intelligence", "machine learning", "knowledge systems", "robotics"},
			Field:       "Artificial Intelligence",
			Deadline:    "2025-01-07",
		},
	}
}

// getStaticArxivCategories 获取静态arXiv分类信息
func (ast *AgentSearchTool) getStaticArxivCategories() []CategoryInfo {
	return []CategoryInfo{
		{
			ID:          "cs.AI",
			Name:        "Artificial Intelligence",
			Description: "Covers all areas of AI except reasoning, planning and knowledge representation",
			Keywords:    []string{"artificial intelligence", "machine learning", "neural networks", "expert systems"},
			Related:     []string{"cs.LG", "cs.CV", "cs.CL"},
		},
		{
			ID:          "cs.LG",
			Name:        "Machine Learning",
			Description: "Machine learning, learning theory, statistical inference",
			Keywords:    []string{"machine learning", "deep learning", "learning theory", "statistical inference"},
			Related:     []string{"cs.AI", "cs.CV", "cs.NE"},
		},
		{
			ID:          "cs.CV",
			Name:        "Computer Vision",
			Description: "Computer vision, pattern recognition, scene understanding",
			Keywords:    []string{"computer vision", "image processing", "pattern recognition", "object detection"},
			Related:     []string{"cs.AI", "cs.LG", "cs.MM"},
		},
		{
			ID:          "cs.CL",
			Name:        "Computation and Language",
			Description: "Natural language processing, computational linguistics",
			Keywords:    []string{"natural language processing", "computational linguistics", "language models", "text mining"},
			Related:     []string{"cs.AI", "cs.IR", "cs.LG"},
		},
		{
			ID:          "cs.RO",
			Name:        "Robotics",
			Description: "Robotics, control, autonomous agents",
			Keywords:    []string{"robotics", "autonomous agents", "control systems", "manipulation"},
			Related:     []string{"cs.AI", "cs.SY", "cs.CV"},
		},
		{
			ID:          "cs.IR",
			Name:        "Information Retrieval",
			Description: "Information retrieval, search engines, recommendation systems",
			Keywords:    []string{"information retrieval", "search engines", "recommendation systems", "data mining"},
			Related:     []string{"cs.CL", "cs.AI", "cs.DB"},
		},
		{
			ID:          "cs.DB",
			Name:        "Databases",
			Description: "Database systems, knowledge management, data mining",
			Keywords:    []string{"databases", "knowledge management", "data mining", "information systems"},
			Related:     []string{"cs.IR", "cs.AI", "cs.LG"},
		},
		{
			ID:          "cs.NE",
			Name:        "Neural and Evolutionary Computing",
			Description: "Neural networks, genetic algorithms, evolutionary computation",
			Keywords:    []string{"neural networks", "genetic algorithms", "evolutionary computation", "deep learning"},
			Related:     []string{"cs.LG", "cs.AI", "cs.CV"},
		},
	}
}


func (ast *AgentSearchTool) getCurrentTrendingKeywords() []string {
	return []string{
		"large language models", "transformers", "diffusion models", "vision transformers",
		"multimodal", "few-shot learning", "prompt engineering", "retrieval-augmented generation",
		"autonomous agents", "reinforcement learning", "graph neural networks", "self-supervised learning",
		"foundation models", "parameter-efficient fine-tuning", "chain-of-thought", "in-context learning",
		"vision-language models", "generative AI", "responsible AI", "AI safety",
		"efficient transformers", "model compression", "knowledge distillation", "neural architecture search",
	}
}


func (ast *AgentSearchTool) getCurrentSeason() string {
	month := time.Now().Month()
	switch {
	case month >= 1 && month <= 2:
		return "winter_submission_season"
	case month >= 3 && month <= 5:
		return "spring_notification_season"
	case month >= 6 && month <= 8:
		return "summer_conference_season"
	case month >= 9 && month <= 10:
		return "fall_submission_season"
	default:
		return "winter_preparation_season"
	}
}



// AnalyzeQuery 分析用户查询并生成增强搜索建议
func (ast *AgentSearchTool) AnalyzeQuery(ctx context.Context, userQuery string) (*EnhancedSearchQuery, error) {
	// 获取搜索上下文
	searchContext, err := ast.GetSearchContext(ctx)
	if err != nil {
		logger.Warn("获取搜索上下文失败: %v", err)
		searchContext = &SearchContext{} // 使用空上下文作为降级
	}

	// 基础查询分析
	enhancedQuery := &EnhancedSearchQuery{
		OriginalQuery: userQuery,
		Context:       searchContext,
	}

	// 查询词规范化
	normalizedQuery := strings.ToLower(userQuery)
	queryTokens := strings.Fields(normalizedQuery)

	// 会议匹配分析
	enhancedQuery.OpenReviewVenue = ast.matchVenue(normalizedQuery, searchContext.AvailableVenues)
	if enhancedQuery.OpenReviewVenue != "" {
		enhancedQuery.RecommendedVenues = []string{enhancedQuery.OpenReviewVenue}
	}

	// 分类匹配分析
	enhancedQuery.RecommendedCategories = ast.matchCategories(queryTokens, searchContext.ArxivCategories)

	// 关键词扩展
	enhancedQuery.ExpandedKeywords = ast.expandKeywords(queryTokens, searchContext.TrendingKeywords)

	// 生成优化的 arXiv 查询
	enhancedQuery.ArxivQuery = ast.buildArxivQuery(userQuery, enhancedQuery.RecommendedCategories, enhancedQuery.ExpandedKeywords)

	// 计算置信度
	enhancedQuery.Confidence = ast.calculateConfidence(enhancedQuery, searchContext)

	// 确定搜索策略
	enhancedQuery.SearchStrategy = ast.determineSearchStrategy(enhancedQuery, searchContext)

	logger.Info("AgentSearchTool: 查询分析完成 - 会议: %s, 分类: %v, 置信度: %.2f",
		enhancedQuery.OpenReviewVenue, enhancedQuery.RecommendedCategories, enhancedQuery.Confidence)

	return enhancedQuery, nil
}

// matchVenue 匹配最相关的会议
func (ast *AgentSearchTool) matchVenue(query string, venues []VenueInfo) string {
	queryTokens := strings.Fields(query)
	bestMatch := ""
	bestScore := 0.0

	for _, venue := range venues {
		score := ast.calculateVenueMatchScore(queryTokens, venue)
		if score > bestScore && score > 0.1 { // 降低最低阈值
			bestScore = score
			bestMatch = venue.ID
		}
	}

	return bestMatch
}

// calculateVenueMatchScore 计算会议匹配分数
func (ast *AgentSearchTool) calculateVenueMatchScore(queryTokens []string, venue VenueInfo) float64 {
	score := 0.0
	queryLower := strings.Join(queryTokens, " ")
	venueNameLower := strings.ToLower(venue.Name)
	venueIDLower := strings.ToLower(venue.ID)

	// 1. 检查精确匹配（会议缩写）
	for _, token := range queryTokens {
		// 检查会议ID中是否包含查询词
		if strings.Contains(venueIDLower, token) {
			score += 2.0 // 给予更高权重
		}
		// 检查会议名称中是否包含查询词
		if strings.Contains(venueNameLower, token) {
			score += 1.5
		}
	}

	// 2. 检查关键词匹配
	for _, token := range queryTokens {
		for _, keyword := range venue.Keywords {
			keywordLower := strings.ToLower(keyword)
			// 精确匹配
			if token == keywordLower {
				score += 1.0
			} else if strings.Contains(keywordLower, token) || strings.Contains(token, keywordLower) {
				// 包含匹配
				score += 0.7
			}
		}
	}

	// 3. 检查领域相关性
	if strings.Contains(queryLower, "machine learning") || strings.Contains(queryLower, "neural") {
		if venue.Field == "Machine Learning" {
			score += 1.0
		}
	}
	if strings.Contains(queryLower, "nlp") || strings.Contains(queryLower, "language") {
		if venue.Field == "Natural Language Processing" {
			score += 1.0
		}
	}
	if strings.Contains(queryLower, "ai") || strings.Contains(queryLower, "intelligence") {
		if venue.Field == "Artificial Intelligence" {
			score += 1.0
		}
	}

	// 4. 基于状态调整分数
	switch venue.Status {
	case "active":
		score *= 1.2
	case "upcoming":
		score *= 1.1
	}

	// 5. 归一化分数
	if len(queryTokens) > 0 {
		score = score / float64(len(queryTokens))
	}

	return score
}

// matchCategories 匹配arXiv分类
func (ast *AgentSearchTool) matchCategories(queryTokens []string, categories []CategoryInfo) []string {
	var matchedCategories []string
	categoryScores := make(map[string]float64)

	// 计算每个分类的匹配分数
	for _, category := range categories {
		score := 0.0
		for _, token := range queryTokens {
			// 检查分类名称和描述
			if strings.Contains(strings.ToLower(category.Name), token) ||
				strings.Contains(strings.ToLower(category.Description), token) {
				score += 1.0
			}

			// 检查关键词
			for _, keyword := range category.Keywords {
				if strings.Contains(strings.ToLower(keyword), token) {
					score += 0.8
				}
			}
		}

		if score > 0.5 { // 设置阈值
			categoryScores[category.ID] = score
		}
	}

	// 按分数排序并返回前3个
	for catID := range categoryScores {
		if len(matchedCategories) < 3 {
			matchedCategories = append(matchedCategories, catID)
		}
	}

	return matchedCategories
}

// expandKeywords 扩展关键词
func (ast *AgentSearchTool) expandKeywords(queryTokens []string, trendingKeywords []string) []string {
	var expanded []string
	usedKeywords := make(map[string]bool)

	// 添加原始查询词
	for _, token := range queryTokens {
		if !usedKeywords[token] {
			expanded = append(expanded, token)
			usedKeywords[token] = true
		}
	}

	// 添加相关的热门关键词
	for _, token := range queryTokens {
		for _, trending := range trendingKeywords {
			if strings.Contains(strings.ToLower(trending), token) && !usedKeywords[strings.ToLower(trending)] {
				expanded = append(expanded, trending)
				usedKeywords[strings.ToLower(trending)] = true
			}
		}
	}

	// 限制扩展关键词数量
	if len(expanded) > 10 {
		expanded = expanded[:10]
	}

	return expanded
}

// buildArxivQuery 构建优化的arXiv查询
func (ast *AgentSearchTool) buildArxivQuery(originalQuery string, categories []string, keywords []string) string {
	var queryParts []string

	// 基础查询：标题和摘要搜索
	if len(keywords) > 0 {
		titleTerms := make([]string, 0)
		absTerms := make([]string, 0)

		// 前5个关键词用于标题搜索
		for i, kw := range keywords {
			if i < 5 {
				titleTerms = append(titleTerms, fmt.Sprintf("ti:\"%s\"", kw))
			}
			if i < 8 {
				absTerms = append(absTerms, fmt.Sprintf("abs:\"%s\"", kw))
			}
		}

		if len(titleTerms) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("(%s)", strings.Join(titleTerms, " OR ")))
		}
		if len(absTerms) > 0 {
			queryParts = append(queryParts, fmt.Sprintf("(%s)", strings.Join(absTerms, " OR ")))
		}
	} else {
		// 降级到原始查询
		queryParts = append(queryParts, fmt.Sprintf("ti:\"%s\" OR abs:\"%s\"", originalQuery, originalQuery))
	}

	// 添加分类过滤
	if len(categories) > 0 {
		catTerms := make([]string, 0, len(categories))
		for _, cat := range categories {
			catTerms = append(catTerms, fmt.Sprintf("cat:%s", cat))
		}
		queryParts = append(queryParts, fmt.Sprintf("(%s)", strings.Join(catTerms, " OR ")))
	}

	// 构建最终查询
	if len(queryParts) == 1 {
		return queryParts[0]
	}
	return strings.Join(queryParts, " AND ")
}

// calculateConfidence 计算置信度
func (ast *AgentSearchTool) calculateConfidence(query *EnhancedSearchQuery, _ *SearchContext) float64 {
	confidence := 0.5 // 基础置信度

	// 如果匹配到会议，提高置信度
	if query.OpenReviewVenue != "" {
		confidence += 0.25
	}

	// 如果匹配到分类，提高置信度
	if len(query.RecommendedCategories) > 0 {
		confidence += 0.15
	}

	// 如果扩展了关键词，提高置信度
	if len(query.ExpandedKeywords) > len(strings.Fields(query.OriginalQuery)) {
		confidence += 0.1
	}

	// 确保置信度在合理范围内
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// determineSearchStrategy 确定搜索策略
func (ast *AgentSearchTool) determineSearchStrategy(query *EnhancedSearchQuery, _ *SearchContext) string {
	if query.OpenReviewVenue != "" {
		if len(query.RecommendedCategories) > 0 {
			return "hybrid_conference_and_category"
		}
		return "conference_focused"
	}

	if len(query.RecommendedCategories) > 0 {
		return "category_focused"
	}

	if query.Confidence > 0.7 {
		return "keyword_optimized"
	}

	return "general_search"
}

// GetSearchSuggestion 获取搜索建议
func (ast *AgentSearchTool) GetSearchSuggestion(_ context.Context, userQuery string) ([]string, error) {
	enhancedQuery, err := ast.AnalyzeQuery(context.Background(), userQuery)
	if err != nil {
		return nil, err
	}

	var suggestions []string

	// 基于搜索策略生成建议
	switch enhancedQuery.SearchStrategy {
	case "conference_focused":
		suggestions = append(suggestions, fmt.Sprintf("建议专注于 %s 会议的最新论文", enhancedQuery.OpenReviewVenue))
	case "category_focused":
		suggestions = append(suggestions, fmt.Sprintf("建议关注 %s 等分类的论文", strings.Join(enhancedQuery.RecommendedCategories, ", ")))
	case "hybrid_conference_and_category":
		suggestions = append(suggestions, fmt.Sprintf("建议结合 %s 会议和 %s 分类进行搜索",
			enhancedQuery.OpenReviewVenue, strings.Join(enhancedQuery.RecommendedCategories, ", ")))
	default:
		suggestions = append(suggestions, "建议使用更具体的关键词以获得更好的搜索结果")
	}

	// 添加当前季节的建议
	if enhancedQuery.Context != nil {
		switch enhancedQuery.Context.CurrentSeason {
		case "winter_submission_season":
			suggestions = append(suggestions, "当前是论文投稿旺季，可以关注最新的研究成果")
		case "summer_conference_season":
			suggestions = append(suggestions, "当前是会议季，可以关注顶级会议的最新论文")
		}
	}

	// 如果有即将到来的截止日期，提供建议
	if enhancedQuery.Context != nil && len(enhancedQuery.Context.UpcomingDeadlines) > 0 {
		deadline := enhancedQuery.Context.UpcomingDeadlines[0]
		suggestions = append(suggestions, fmt.Sprintf("即将截止: %s (%s)", deadline.VenueName, deadline.Deadline))
	}

	return suggestions, nil
}

// ExportSearchContext 导出搜索上下文为JSON（用于调试）
func (ast *AgentSearchTool) ExportSearchContext(_ context.Context) (string, error) {
	searchContext, err := ast.GetSearchContext(context.Background())
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(searchContext, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化搜索上下文失败: %w", err)
	}

	return string(data), nil
}