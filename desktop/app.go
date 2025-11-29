package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"PaperHunter/config"
	"PaperHunter/internal/core"
	"PaperHunter/internal/hyde"
	"PaperHunter/internal/models"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"

	"github.com/cloudwego/eino/adk"
)

type App struct {
	ctx          context.Context
	coreApp      *core.App
	logfile      string
	config       *config.AppConfig
	crawlService *CrawlService
	agent        adk.Agent        // Agent 实例
	searchTool   *AgentSearchTool // AgentSearchTool 实例
	hydeSvc      hyde.Service     // HyDE 服务（用于生成虚拟论文）
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.initLogger()
	a.initConfig()

	a.initCoreApp()
	a.initHyDE()
	a.initSearchTool()
	a.initAgent()
}

func (a *App) initHyDE() {
	if a.config == nil {
		logger.Warn("配置未初始化，跳过 HyDE 服务初始化")
		return
	}

	svc, err := hyde.New(a.config.LLM)
	if err != nil {
		logger.Error("HyDE 服务初始化失败: %v", err)
		return
	}

	a.hydeSvc = svc
	logger.Info("HyDE 服务初始化成功")
}

func (a *App) initConfig() {
	homeDir, _ := os.UserHomeDir()
	configFilePath := filepath.Join(homeDir, ".quicksearch", "config", "config.yaml")

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		logger.Info("配置文件不存在，正在创建示例配置文件: %s", configFilePath)
		if err := config.CreateExampleConfig(); err != nil {
			logger.Error("创建示例配置文件失败: %v", err)

		} else {
			logger.Info("已创建示例配置文件，请根据需要编辑配置文件")
		}
	} else if err != nil {
		logger.Warn("检查配置文件时出错: %v，将使用默认配置", err)
	}

	cfg, err := config.Init("")
	if err != nil {
		logger.Error("加载配置失败: %v", err)

		cfg, _ = config.Init("")
	}

	a.config = cfg
	if cfg != nil {
		logger.Info("配置加载成功，配置文件路径: %s", config.GetConfigPath())
	}
}

func (a *App) initLogger() {
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".quicksearch", "logs")

	os.MkdirAll(logDir, 0755)
	layout := "200601021504"

	now := time.Now().Format(layout)

	a.logfile = filepath.Join(logDir, now+".log")

	logger.InitWithFile("INFO", false, a.logfile)

	logger.Info("桌面应用启动，日志文件: %s", a.logfile)
}

func (a *App) initCoreApp() {
	// 确保配置已初始化
	if a.config == nil {
		logger.Error("配置未初始化，无法启动核心模块")
		return
	}

	cfg := a.config

	

	var err error
	a.coreApp, err = core.NewApp(cfg.Database.Path, cfg.Embedder,
		map[string]platform.Config{
			"arxiv":      &cfg.Arxiv,
			"openreview": &cfg.OpenReview,
			"acl":        &cfg.ACL,
			"ssrn":       &cfg.SSRN,
		}, cfg.Zotero, cfg.FeiShu)

	if err != nil {
		logger.Error("初始化核心模块失败: %v", err)
	} else {
		logger.Info("核心模块启动成功")
	}
}

func (a *App) initSearchTool() {
	a.searchTool = NewAgentSearchTool()
	if a.searchTool != nil {
		logger.Info("AgentSearchTool 初始化成功")
	} else {
		logger.Error("AgentSearchTool 初始化失败")
	}
}

func (a *App) initAgent() {
	// 只有在核心模块初始化成功后才初始化 agent
	if a.coreApp == nil {
		logger.Warn("核心模块未初始化，跳过 agent 初始化")
		return
	}

	// 初始化 agent
	agent := NewPaperAgent(a)
	if agent == nil {
		logger.Warn("Agent 初始化失败，某些功能可能不可用")
	} else {
		a.agent = agent
		logger.Info("Agent 初始化成功")
	}
}

func (a *App) SetLogLevel(level string) {
	logger.SetLevel(level)
	logger.Info("日志级别已设置为: %s", level)
}

func (a *App) CrawlPapers(platform string, params map[string]interface{}) (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("core app not initialized")
	}

	if a.crawlService == nil {
		a.crawlService = NewCrawlService(a)
	}

	taskID, err := a.crawlService.StartCrawl(platform, params)
	if err != nil {
		return "", fmt.Errorf("failed to start crawl task: %w", err)
	}

	logger.Info("Started crawl task: %s for platform: %s", taskID, platform)
	return taskID, nil
}

// GetCrawlTask 获取爬取任务状态
func (a *App) GetCrawlTask(taskID string) (string, error) {
	if a.crawlService == nil {
		return "", fmt.Errorf("crawl service not initialized")
	}

	task, err := a.crawlService.GetTask(taskID)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}

	return string(data), nil
}

func (a *App) GetCrawlTaskLogs(taskID string) (string, error) {
	if a.crawlService == nil {
		return "", fmt.Errorf("crawl service not initialized")
	}

	logs, err := a.crawlService.GetTaskLogs(taskID)
	if err != nil {
		return "", err
	}

	// 序列化为JSON
	data, err := json.Marshal(logs)
	if err != nil {
		return "", fmt.Errorf("failed to marshal logs: %w", err)
	}

	return string(data), nil
}

func (a *App) ExportSelection(format string, source string, ids []string, output string, feishuName string, collection string) (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("core app not initialized")
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no papers selected")
	}

	var conditions []string
	var params []interface{}

	if source != "" {
		// 生成条件: source = ? AND source_id IN (?,...,?)
		placeholders := make([]string, 0, len(ids))
		for range ids {
			placeholders = append(placeholders, "?")
		}
		params = append(params, source)
		for _, id := range ids {
			params = append(params, id)
		}
		conditions = []string{"source = ?", fmt.Sprintf("source_id IN (%s)", strings.Join(placeholders, ","))}
	} else {
		// source 为空时，只使用 source_id 条件（但这种情况不应该发生）
		placeholders := make([]string, 0, len(ids))
		for range ids {
			placeholders = append(placeholders, "?")
		}
		for _, id := range ids {
			params = append(params, id)
		}
		conditions = []string{fmt.Sprintf("source_id IN (%s)", strings.Join(placeholders, ","))}
	}

	ctx := context.Background()
	switch strings.ToLower(format) {
	case "csv", "json":
		if output == "" {
			now := time.Now().Format("20060102_150405")
			output = fmt.Sprintf("selection_%s.%s", now, format)
		}
		return output, a.coreApp.ExportPapers(ctx, format, output, conditions, params, 0)
	case "zotero":
		return "", a.coreApp.ExportToZotero(ctx, collection, conditions, params, 0)
	case "feishu":
		name := feishuName
		if name == "" {
			name = "Papers"
		}
		url, err := a.coreApp.ExportToFeiShuBitableWithURL(ctx, name, name, conditions, params, 0)
		return url, err
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportSelectionByPapers 按论文列表导出，支持多 source（通过传入完整的 source+id 对）
// 用于前端传递完整的论文信息，避免 source 不匹配的问题
func (a *App) ExportSelectionByPapers(format string, paperPairs []map[string]string, output string, feishuName string, collection string) (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("core app not initialized")
	}
	if len(paperPairs) == 0 {
		return "", fmt.Errorf("no papers selected")
	}

	// 按 source 分组
	sourceGroups := make(map[string][]string)
	for _, pair := range paperPairs {
		source := pair["source"]
		id := pair["id"]
		if source == "" || id == "" {
			continue
		}
		sourceGroups[source] = append(sourceGroups[source], id)
	}

	// 构建 OR 条件: (source = 'arxiv' AND source_id IN (...)) OR (source = 'ssrn' AND source_id IN (...))
	var conditionParts []string
	var params []interface{}

	for source, ids := range sourceGroups {
		placeholders := make([]string, 0, len(ids))
		for range ids {
			placeholders = append(placeholders, "?")
		}
		conditionParts = append(conditionParts, fmt.Sprintf("(source = ? AND source_id IN (%s))", strings.Join(placeholders, ",")))
		params = append(params, source)
		for _, id := range ids {
			params = append(params, id)
		}
	}

	if len(conditionParts) == 0 {
		return "", fmt.Errorf("no valid papers selected")
	}

	conditions := []string{fmt.Sprintf("(%s)", strings.Join(conditionParts, " OR "))}

	ctx := context.Background()
	switch strings.ToLower(format) {
	case "csv", "json":
		if output == "" {
			now := time.Now().Format("20060102_150405")
			output = fmt.Sprintf("selection_%s.%s", now, format)
		}
		return output, a.coreApp.ExportPapers(ctx, format, output, conditions, params, 0)
	case "zotero":
		return "", a.coreApp.ExportToZotero(ctx, collection, conditions, params, 0)
	case "feishu":
		name := feishuName
		if name == "" {
			name = "Papers"
		}
		url, err := a.coreApp.ExportToFeiShuBitableWithURL(ctx, name, name, conditions, params, 0)
		return url, err
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// AnalyzeSearchQuery 使用 AgentSearchTool 分析搜索查询
func (a *App) AnalyzeSearchQuery(userQuery string) (string, error) {
	if a.searchTool == nil {
		return "", fmt.Errorf("AgentSearchTool not initialized")
	}

	ctx := context.Background()
	enhancedQuery, err := a.searchTool.AnalyzeQuery(ctx, userQuery)
	if err != nil {
		return "", fmt.Errorf("failed to analyze query: %w", err)
	}

	// 序列化结果
	data, err := json.MarshalIndent(enhancedQuery, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(data), nil
}

// GetSearchSuggestions 获取搜索建议
func (a *App) GetSearchSuggestions(userQuery string) (string, error) {
	if a.searchTool == nil {
		return "", fmt.Errorf("AgentSearchTool not initialized")
	}

	ctx := context.Background()
	suggestions, err := a.searchTool.GetSearchSuggestion(ctx, userQuery)
	if err != nil {
		return "", fmt.Errorf("failed to get suggestions: %w", err)
	}

	// 序列化结果
	result := map[string]interface{}{
		"query":       userQuery,
		"suggestions": suggestions,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal suggestions: %w", err)
	}

	return string(data), nil
}

// GetSearchContext 获取搜索上下文信息（用于调试和展示）
func (a *App) GetSearchContext() (string, error) {
	if a.searchTool == nil {
		return "", fmt.Errorf("AgentSearchTool not initialized")
	}

	ctx := context.Background()
	context, err := a.searchTool.ExportSearchContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get search context: %w", err)
	}

	return context, nil
}

// DebugRecommendationInfo 调试推荐信息
func (a *App) DebugRecommendationInfo() (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("core app not initialized")
	}

	ctx := context.Background()
	debugInfo := make(map[string]interface{})

	// 1. 检查 Zotero 论文数量
	zoteroPapers, err := getZoteroPapers("", 100) // 不限制集合
	if err != nil {
		debugInfo["zotero_error"] = err.Error()
	} else {
		debugInfo["zotero_paper_count"] = len(zoteroPapers)
		debugInfo["zotero_papers"] = make([]map[string]interface{}, 0, len(zoteroPapers))
		for i, paper := range zoteroPapers {
			if i >= 10 { // 只显示前10篇
				break
			}
			debugInfo["zotero_papers"] = append(debugInfo["zotero_papers"].([]map[string]interface{}), map[string]interface{}{
				"title":        paper.Title,
				"source":       paper.Source,
				"source_id":    paper.SourceID,
				"categories":   paper.Categories,
				"abstract_len": len(paper.Abstract),
			})
		}
	}

	// 2. 检查数据库中的论文数量
	today := time.Now()
	startDate := time.Date(today.Year(), today.Month(), today.Day()-7, 0, 0, 0, 0, today.Location())
	endDate := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 999999999, today.Location())

	// 统计不同平台的论文数量
	platformCounts := make(map[string]int)
	totalCount := 0

	platforms := []string{"arxiv", "openreview", "acl", "ssrn"}
	for _, platform := range platforms {
		cond := models.SearchCondition{
			Sources:  []string{platform},
			DateFrom: &startDate,
			DateTo:   &endDate,
			Limit:    1000,
		}

		results, err := a.coreApp.Search(ctx, core.SearchOptions{
			Condition: cond,
			Semantic:  false,
		})

		if err != nil {
			debugInfo[platform+"_error"] = err.Error()
		} else {
			platformCounts[platform] = len(results)
			totalCount += len(results)
		}
	}

	debugInfo["platform_counts"] = platformCounts
	debugInfo["total_recent_papers"] = totalCount

	// 3. 检查缓存状态
	if a.searchTool != nil {
		cacheInfo := make(map[string]interface{})
		cacheInfo["cache_entries"] = len(a.searchTool.cache)

		expiredCount := 0
		now := time.Now()
		for key, entry := range a.searchTool.cache {
			if entry.ExpiresAt.Before(now) {
				expiredCount++
				_ = key // 避免未使用变量警告
			}
		}
		cacheInfo["expired_entries"] = expiredCount
		debugInfo["search_tool_cache"] = cacheInfo
	}

	// 4. 生成修复建议
	suggestions := make([]string, 0)

	if len(zoteroPapers) == 0 {
		suggestions = append(suggestions, "📝 Zotero 库为空，建议添加一些种子论文以获得个性化推荐")
	}

	if totalCount == 0 {
		suggestions = append(suggestions, "📅 数据库中没有最近论文，建议先进行论文爬取")
	}

	if totalCount < 100 {
		suggestions = append(suggestions, "📊 数据库中论文数量较少，建议扩大爬取范围")
	}

	debugInfo["suggestions"] = suggestions

	// 转换为JSON
	data, err := json.MarshalIndent(debugInfo, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal debug info failed: %w", err)
	}

	return string(data), nil
}
