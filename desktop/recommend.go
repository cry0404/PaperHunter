package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"PaperHunter/internal/core"
	"PaperHunter/internal/models"
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


type AgentLogEntry struct {
	Type      string `json:"type"`      // "user", "assistant", "tool_call", "tool_result"
	Content   string `json:"content"`   // 消息内容
	Timestamp string `json:"timestamp"` // 时间戳
}


type RecommendResult struct {
	CrawledToday    bool                  `json:"crawledToday"`
	ArxivCrawlCount int                   `json:"arxivCrawlCount"`
	SeedPaperCount  int                   `json:"seedPaperCount"`
	Recommendations []RecommendationGroup `json:"recommendations"`
	Message         string                `json:"message"`
	AgentLogs       []AgentLogEntry       `json:"agentLogs"`
}


type UserIntent struct {
	GeneratedTitle    string `json:"generated_title"`
	GeneratedAbstract string `json:"generated_abstract"`
}


func (a *App) logAndEmit(log AgentLogEntry) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "agent-log", log)
	}
}

// generateHypotheticalPaperWithHyDE 使用 HyDE 服务生成虚拟论文
func (a *App) generateHypotheticalPaperWithHyDE(userQuery string) (string, string, error) {
	fallback := func() (string, string, error) {
		title := strings.TrimSpace(userQuery)
		if title == "" {
			title = "generic research topic"
		}
		abstract := title
		return title, abstract, nil
	}

	if a.hydeSvc == nil {
		logger.Warn("HyDE 服务未初始化，使用降级结果")
		return fallback()
	}

	if userQuery == "" {
		return fallback()
	}

	ctx := context.Background()
	paper, err := a.hydeSvc.GenerateHypotheticalPaper(ctx, userQuery)
	if err != nil {
		logger.Warn("HyDE 生成失败，使用降级结果: %v", err)
		return fallback()
	}

	if paper == nil {
		logger.Warn("HyDE 返回空结果，使用降级结果")
		return fallback()
	}

	return paper.Title, paper.Abstract, nil
}

// analyzeUserIntent 使用 HyDE，并优先利用关键词检索得到的 topK 文章上下文
func (a *App) analyzeUserIntent(opts RecommendOptions, dateFrom, dateTo string, keywordTopK int) (*UserIntent, []AgentLogEntry, error) {
	logs := make([]AgentLogEntry, 0)
	userQuery := strings.TrimSpace(opts.InterestQuery)
	if userQuery == "" {
		return nil, logs, nil
	}

	logger.Info("分析用户意图 (HyDE with keyword topK=%d): %s", keywordTopK, userQuery)

	ctx := context.Background()

	// 根据已经下载的，先选取最相近的几篇生成的 hyde 做推荐
	var fromDate, toDatePtr *time.Time
	if dateFrom != "" {
		if from, err := time.Parse("2006-01-02", dateFrom); err == nil {
			tmp := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
			fromDate = &tmp
		}
	}
	if dateTo != "" {
		if to, err := time.Parse("2006-01-02", dateTo); err == nil {
			tmp := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
			toDatePtr = &tmp
		}
	}

	// 关键词预检：限定 arXiv，TopK=keywordTopK
	var hydeInput = userQuery
	if a.coreApp != nil {
		cond := models.SearchCondition{
			Limit:    keywordTopK,
			Sources:  []string{"arxiv"},
			DateFrom: fromDate,
			DateTo:   toDatePtr,
		}
		searchOpts := core.SearchOptions{
			Query:     userQuery,
			Condition: cond,
			TopK:      keywordTopK,
			Semantic:  false,
		}

		results, err := a.coreApp.Search(ctx, searchOpts)
		if err != nil {
			logger.Warn("关键词预检失败，使用原始查询作为 HyDE 输入: %v", err)
		} else if len(results) > 0 {
			builder := strings.Builder{}
			builder.WriteString("User query: ")
			builder.WriteString(userQuery)
			builder.WriteString("\n\nTop related arXiv papers (title + abstract):\n")
			for i, sp := range results {
				if i >= keywordTopK {
					break
				}
				builder.WriteString(fmt.Sprintf("%d) %s\n", i+1, strings.TrimSpace(sp.Paper.Title)))
				builder.WriteString(truncateText(strings.TrimSpace(sp.Paper.Abstract), 800))
				builder.WriteString("\n\n")
			}
			hydeInput = builder.String()
		}
	}

	// 使用 HyDE 服务生成虚拟论文
	generatedTitle, generatedAbstract, err := a.generateHypotheticalPaperWithHyDE(hydeInput)
	if err != nil {
		logger.Warn("HyDE 生成失败，请根据日志调整: %v", err)
		// 回退到简单方案
		return nil, logs, nil
	}

	intent := &UserIntent{
		GeneratedTitle:    generatedTitle,
		GeneratedAbstract: generatedAbstract,
	}

	logs = append(logs, AgentLogEntry{
		Type:      "tool_result",
		Content:   fmt.Sprintf("Generated paper (HyDE with top%d): %s\n%s", keywordTopK, generatedTitle, generatedAbstract),
		Timestamp: time.Now().Format(time.RFC3339),
	})

	return intent, logs, nil
}

func truncateText(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// GetDailyRecommendations 获取每日推荐
func (a *App) GetDailyRecommendations(opts RecommendOptions) (string, error) {
	logger.Info("开始获取每日推荐 - 兴趣: %s", opts.InterestQuery)

	var agentLogs []AgentLogEntry

	// 记录开始
	agentLogs = append(agentLogs, AgentLogEntry{
		Type:      "user",
		Content:   opts.InterestQuery,
		Timestamp: time.Now().Format(time.RFC3339),
	})

	if opts.LocalFilePath != "" {
		agentLogs = append(agentLogs, AgentLogEntry{
			Type:      "tool_call",
			Content:   fmt.Sprintf("Import local file: %s", opts.LocalFilePath),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}


	result, err := a.getDailyRecommendationsDirect(opts, agentLogs)
	if err != nil {
		// 记录错误
		agentLogs = append(agentLogs, AgentLogEntry{
			Type:      "error",
			Content:   fmt.Sprintf("Recommendation failed: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})

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
