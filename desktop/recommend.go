package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		logger.Warn("HyDE 生成失败，请根据日志调整: %v", err)
		// 回退到简单方案
		return nil, nil
	}

	return &UserIntent{
		GeneratedTitle:    generatedTitle,
		GeneratedAbstract: generatedAbstract,
	}, nil
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


	intent, err := a.analyzeUserIntent(opts.InterestQuery)
	if err != nil {
		logger.Error("意图分析失败: %v", err)
		return "", fmt.Errorf("intent analysis failed: %w", err)
	}

	agentLogs = append(agentLogs, AgentLogEntry{
		Type:      "tool_result",
		Content:   fmt.Sprintf("Generated paper: %s\n%s", intent.GeneratedTitle, intent.GeneratedAbstract),
		Timestamp: time.Now().Format(time.RFC3339),
	})

	result, err := a.getDailyRecommendationsDirect(opts, agentLogs, intent)
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
