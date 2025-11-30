package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"PaperHunter/pkg/logger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RecommendOptions 推荐选项
type RecommendOptions struct {
	InterestQuery      string   `json:"interestQuery"`      // 用户输入的兴趣查询
	Platforms          []string `json:"platforms"`          // 要爬取的平台列表
	ZoteroCollection   string   `json:"zoteroCollection"`   // Zotero collection key（可选）
	TopK               int      `json:"topK"`               // 推荐数量
	MaxRecommendations int      `json:"maxRecommendations"` // 最大推荐总数
	ForceCrawl         bool     `json:"forceCrawl"`         // 强制重新爬取
	DateFrom           string   `json:"dateFrom"`           // 开始日期 YYYY-MM-DD
	DateTo             string   `json:"dateTo"`             // 结束日期 YYYY-MM-DD
	LocalFilePath      string   `json:"localFilePath"`      // 本地文件路径
	LocalFileAction    string   `json:"localFileAction"`    // 本地文件操作
}

// AgentLogEntry Agent 日志条目（简化版）
type AgentLogEntry struct {
	Type      string `json:"type"`      // "user", "assistant", "tool_call", "tool_result"
	Content   string `json:"content"`   // 消息内容
	Timestamp string `json:"timestamp"` // 时间戳
}

// RecommendResult 推荐结果（适配前端格式）
type RecommendResult struct {
	CrawledToday    bool                  `json:"crawledToday"`
	ArxivCrawlCount int                   `json:"arxivCrawlCount"`
	SeedPaperCount  int                   `json:"seedPaperCount"`
	Recommendations []RecommendationGroup `json:"recommendations"`
	Message         string                `json:"message"`
	AgentLogs       []AgentLogEntry       `json:"agentLogs"`
}

// UserIntent 简化的用户意图分析结果
type UserIntent struct {
	GeneratedTitle    string `json:"generated_title"`
	GeneratedAbstract string `json:"generated_abstract"`
}

// logAndEmit 辅助函数：记录日志并发送事件
func (a *App) logAndEmit(log AgentLogEntry) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "agent-log", log)
	}
}

// analyzeUserIntent 使用 HyDE 分析用户意图并生成虚拟论文
func (a *App) analyzeUserIntent(userQuery string) (*UserIntent, error) {
	logger.Info("分析用户意图 (HyDE): %s", userQuery)

	// 使用 HyDE 服务生成虚拟论文
	generatedTitle, generatedAbstract, err := a.generateHypotheticalPaperWithHyDE(userQuery)
	if err != nil {
		logger.Warn("HyDE 生成失败，使用回退方案: %v", err)
		// 回退到简单方案
		generatedTitle, generatedAbstract = a.generateHypotheticalPaperFallback(userQuery)
	}

	return &UserIntent{
		GeneratedTitle:    generatedTitle,
		GeneratedAbstract: generatedAbstract,
	}, nil
}

// generateHypotheticalPaperWithHyDE 使用 HyDE 服务生成虚拟论文
func (a *App) generateHypotheticalPaperWithHyDE(userQuery string) (string, string, error) {
	if a.hydeSvc == nil {
		return "", "", fmt.Errorf("HyDE 服务未初始化")
	}

	if userQuery == "" {
		return "", "", fmt.Errorf("用户查询为空")
	}

	ctx := context.Background()
	paper, err := a.hydeSvc.GenerateHypotheticalPaper(ctx, userQuery)
	if err != nil {
		return "", "", fmt.Errorf("HyDE 生成失败: %w", err)
	}

	return paper.Title, paper.Abstract, nil
}

// generateHypotheticalPaperFallback 回退方案：简单的字符串生成
func (a *App) generateHypotheticalPaperFallback(userQuery string) (string, string) {
	if userQuery == "" {
		return "Recent Advances in Computer Science Research",
			"This paper surveys recent developments in computer science, covering emerging trends in machine learning, natural language processing, and systems research. We analyze state-of-the-art approaches and discuss future research directions."
	}

	// 提取关键词
	keywords := extractQueryKeywords(userQuery)
	keywordStr := userQuery
	if len(keywords) > 0 {
		keywordStr = joinKeywords(keywords)
	}

	title := fmt.Sprintf("Advances in %s: Methods, Applications and Future Directions", capitalizeFirst(keywordStr))
	abstract := fmt.Sprintf(`This paper presents a comprehensive study on %s, addressing key challenges and proposing novel solutions in the field. We first analyze the current state of research and identify critical gaps in existing approaches. Our work introduces innovative methodologies that significantly improve upon baseline methods, demonstrating strong performance across multiple benchmarks. Through extensive experiments, we validate our approach and show substantial improvements in both efficiency and effectiveness. The proposed techniques offer practical benefits for real-world applications while maintaining theoretical rigor. We also discuss limitations and outline promising directions for future research in this rapidly evolving area.`, keywordStr)

	return title, abstract
}

// extractQueryKeywords 从查询中提取关键词
func extractQueryKeywords(query string) []string {
	// 简单的关键词提取：按空格分割，过滤停用词
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "and": true, "or": true, "with": true, "by": true,
		"about": true, "research": true, "paper": true, "papers": true,
		"related": true, "want": true, "find": true, "search": true,
		"looking": true, "interested": true, "i": true, "me": true,
		"my": true, "recent": true, "new": true, "latest": true,
		"我": true, "想": true, "看": true, "找": true, "相关": true,
		"论文": true, "最近": true, "的": true, "有关": true,
	}

	words := splitWords(query)
	var keywords []string
	for _, w := range words {
		w = cleanWord(w)
		if len(w) > 1 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}

	return keywords
}

// splitWords 分割单词（支持中英文）
func splitWords(s string) []string {
	var words []string
	var current []rune

	for _, r := range s {
		if r == ' ' || r == ',' || r == '.' || r == '!' || r == '?' || r == ';' || r == ':' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words
}

// cleanWord 清理单词
func cleanWord(w string) string {
	// 转小写并去除首尾标点
	w = toLowerASCII(w)
	for len(w) > 0 && isPunct(rune(w[0])) {
		w = w[1:]
	}
	for len(w) > 0 && isPunct(rune(w[len(w)-1])) {
		w = w[:len(w)-1]
	}
	return w
}

// toLowerASCII 转小写（仅ASCII）
func toLowerASCII(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// isPunct 判断是否是标点
func isPunct(r rune) bool {
	return r == '.' || r == ',' || r == '!' || r == '?' || r == ';' || r == ':' || r == '"' || r == '\''
}

// joinKeywords 连接关键词
func joinKeywords(keywords []string) string {
	if len(keywords) == 0 {
		return ""
	}
	if len(keywords) == 1 {
		return keywords[0]
	}
	if len(keywords) == 2 {
		return keywords[0] + " and " + keywords[1]
	}
	// 取前3个
	if len(keywords) > 3 {
		keywords = keywords[:3]
	}
	result := keywords[0]
	for i := 1; i < len(keywords)-1; i++ {
		result += ", " + keywords[i]
	}
	result += " and " + keywords[len(keywords)-1]
	return result
}

// capitalizeFirst 首字母大写
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] = r[0] - 32
	}
	return string(r)
}

// GetDailyRecommendations 获取每日推荐（简化版）
func (a *App) GetDailyRecommendations(opts RecommendOptions) (string, error) {
	logger.Info("开始获取每日推荐 - 兴趣: %s", opts.InterestQuery)

	var agentLogs []AgentLogEntry

	// 记录开始
	agentLogs = append(agentLogs, AgentLogEntry{
		Type:      "user",
		Content:   opts.InterestQuery,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 如果有本地文件，记录文件导入
	if opts.LocalFilePath != "" {
		agentLogs = append(agentLogs, AgentLogEntry{
			Type:      "tool_call",
			Content:   fmt.Sprintf("Import local file: %s", opts.LocalFilePath),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// 简化的意图分析
	intent, err := a.analyzeUserIntent(opts.InterestQuery)
	if err != nil {
		logger.Error("意图分析失败: %v", err)
		return "", fmt.Errorf("intent analysis failed: %w", err)
	}

	// 记录生成的虚拟论文
	agentLogs = append(agentLogs, AgentLogEntry{
		Type:      "tool_result",
		Content:   fmt.Sprintf("Generated paper: %s\n%s", intent.GeneratedTitle, intent.GeneratedAbstract),
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 使用简化的推荐逻辑
	result, err := a.getDailyRecommendationsDirect(opts, agentLogs, intent)
	if err != nil {
		// 记录错误
		agentLogs = append(agentLogs, AgentLogEntry{
			Type:      "error",
			Content:   fmt.Sprintf("Recommendation failed: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})

		// 返回错误但包含日志
		errorResult := RecommendResult{
			CrawledToday:    false,
			ArxivCrawlCount: 0,
			SeedPaperCount:  0,
			Recommendations: []RecommendationGroup{},
			Message:         err.Error(),
			AgentLogs:       agentLogs,
		}

		errorJson, _ := json.Marshal(errorResult)
		return string(errorJson), nil
	}

	// 记录成功
	agentLogs = append(agentLogs, AgentLogEntry{
		Type:      "assistant",
		Content:   "Successfully generated recommendations",
		Timestamp: time.Now().Format(time.RFC3339),
	})

	// 解析结果并添加日志
	var finalResult RecommendResult
	if err := json.Unmarshal([]byte(result), &finalResult); err != nil {
		logger.Error("解析推荐结果失败: %v", err)
		// 使用默认结果
		finalResult = RecommendResult{
			CrawledToday:    false,
			ArxivCrawlCount: 0,
			SeedPaperCount:  0,
			Recommendations: []RecommendationGroup{},
			Message:         "Failed to parse recommendation result",
			AgentLogs:       agentLogs,
		}
	}

	// 添加最新的日志
	finalResult.AgentLogs = agentLogs

	// 序列化最终结果
	finalJson, err := json.Marshal(finalResult)
	if err != nil {
		logger.Error("序列化最终结果失败: %v", err)
		return "", fmt.Errorf("failed to serialize final result: %w", err)
	}

	logger.Info("推荐完成，返回结果")
	return string(finalJson), nil
}
