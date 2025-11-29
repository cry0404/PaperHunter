package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"

	"github.com/cloudwego/eino/adk"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RecommendOptions 推荐选项（与 DailyRecommendInput 对应）
type RecommendOptions struct {
	InterestQuery      string   `json:"interestQuery"`      // 用户输入的兴趣查询（可选）
	Platforms          []string `json:"platforms"`          // 要爬取的平台列表
	ZoteroCollection   string   `json:"zoteroCollection"`   // Zotero collection key（可选）
	TopK               int      `json:"topK"`               // 每个 Zotero 论文推荐的数量
	MaxRecommendations int      `json:"maxRecommendations"` // 最大推荐总数
	ForceCrawl         bool     `json:"forceCrawl"`         // 强制重新爬取
	DateFrom           string   `json:"dateFrom"`           // 开始日期 YYYY-MM-DD（可选，默认今天）
	DateTo             string   `json:"dateTo"`             // 结束日期 YYYY-MM-DD（可选，默认今天）
}

// AgentLogEntry Agent 日志条目
type AgentLogEntry struct {
	Type      string `json:"type"`      // "user", "assistant", "tool_call", "tool_result"
	Content   string `json:"content"`   // 消息内容
	Timestamp string `json:"timestamp"` // 时间戳
}

// RecommendResult 推荐结果（适配前端格式）
type RecommendResult struct {
	CrawledToday     bool                  `json:"crawledToday"`
	CrawlCount       int                   `json:"crawlCount"`
	ZoteroPaperCount int                   `json:"zoteroPaperCount"`
	Recommendations  []RecommendationGroup `json:"recommendations"` // RecommendationGroup 定义在 zoteroRecommendTool.go 中
	Message          string                `json:"message"`
	AgentLogs        []AgentLogEntry       `json:"agentLogs"` // Agent 交互日志
}

// logAndEmit 辅助函数：记录日志并发送事件
func (a *App) logAndEmit(log AgentLogEntry) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "agent-log", log)
	}
}

// GetDailyRecommendations 获取每日推荐（通过 agent 调用，收集 LLM 日志）
func (a *App) GetDailyRecommendations(opts RecommendOptions) (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("app not initialized")
	}

	if a.agent == nil {
		return "", fmt.Errorf("agent not initialized")
	}

	ctx := context.Background()

	// 构建工具调用参数
	toolParams := map[string]interface{}{
		"action": "daily_recommend",
	}
	if len(opts.Platforms) > 0 {
		toolParams["platforms"] = opts.Platforms
	}
	if opts.ZoteroCollection != "" {
		toolParams["collection_key"] = opts.ZoteroCollection
	}
	if opts.TopK > 0 {
		toolParams["top_k"] = opts.TopK
	}
	if opts.MaxRecommendations > 0 {
		toolParams["max_recommendations"] = opts.MaxRecommendations
	}
	if opts.ForceCrawl {
		toolParams["force_crawl"] = true
	}
	if opts.DateFrom != "" {
		toolParams["date_from"] = opts.DateFrom
	}
	if opts.DateTo != "" {
		toolParams["date_to"] = opts.DateTo
	}

	// 使用 Runner 调用 agent，收集日志
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: a.agent,
	})

	// 构建包含用户兴趣和工具调用的查询
	var query string
	if opts.InterestQuery != "" {
		// 用户提供了兴趣关键词
		query = fmt.Sprintf(`用户感兴趣的主题：%s
请使用 zotero_recommend 工具（action: daily_recommend）获取每日推荐。
参数：%s`, opts.InterestQuery, formatToolParams(toolParams))
	} else {
		// 用户没有提供关键词，基于 Zotero 推荐
		query = fmt.Sprintf(`请使用 zotero_recommend 工具（action: daily_recommend）基于用户 Zotero 库中的论文获取每日推荐。
参数：%s`, formatToolParams(toolParams))
	}

	agentLogs := make([]AgentLogEntry, 0)
	initialLog := AgentLogEntry{
		Type:      "user",
		Content:   query,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	agentLogs = append(agentLogs, initialLog)
	a.logAndEmit(initialLog)

	// 调用 agent, 开始基于 query 查询，这里可以引入 Hype 部分来对于用户的询问作优化
	logger.Info("开始调用 agent，查询: %s", query)
	iter := runner.Query(ctx, query)
	var finalOutput *ZoteroRecommendOutput
	eventCount := 0

	for {
		event, ok := iter.Next()
		if !ok {
			logger.Info("Agent 事件流结束，共处理 %d 个事件", eventCount)
			break
		}

		eventCount++
		logger.Debug("收到 agent 事件 #%d", eventCount)

		if event.Err != nil {
			logger.Warn("Agent 事件错误: %v", event.Err)
			errLog := AgentLogEntry{
				Type:      "error",
				Content:   fmt.Sprintf("错误: %v", event.Err),
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			}
			agentLogs = append(agentLogs, errLog)
			a.logAndEmit(errLog)
			// 不直接返回错误，继续执行回退逻辑
			break
		}

		// 记录所有输出类型
		if event.Output != nil {
			logger.Debug("处理 agent 输出事件")
			// 尝试序列化整个输出以查看结构
			outputJSON, err := json.Marshal(event.Output)
			if err != nil {
				logger.Warn("序列化 agent 输出失败: %v", err)
				// 记录原始输出
				logEntry := AgentLogEntry{
					Type:      "assistant",
					Content:   fmt.Sprintf("输出序列化失败: %v", err),
					Timestamp: time.Now().Format("2006-01-02 15:04:05"),
				}
				agentLogs = append(agentLogs, logEntry)
				a.logAndEmit(logEntry)
			} else {
				var outputMap map[string]interface{}
				if err := json.Unmarshal(outputJSON, &outputMap); err != nil {
					logger.Warn("解析 agent 输出 JSON 失败: %v", err)
					logEntry := AgentLogEntry{
						Type:      "assistant",
						Content:   fmt.Sprintf("输出解析失败: %v\n原始输出: %s", err, string(outputJSON)),
						Timestamp: time.Now().Format("2006-01-02 15:04:05"),
					}
					agentLogs = append(agentLogs, logEntry)
					a.logAndEmit(logEntry)
				} else {
					// 调试：输出所有 keys 和原始 JSON
					keys := make([]string, 0, len(outputMap))
					for k := range outputMap {
						keys = append(keys, k)
					}
					logger.Info("Agent 输出的 keys: %v", keys)
					// 输出原始 JSON 前 500 字符用于调试
					rawJSON := string(outputJSON)
					if len(rawJSON) > 500 {
						rawJSON = rawJSON[:500] + "..."
					}
					logger.Info("Agent 原始输出: %s", rawJSON)

					// 记录输出类型（尝试多种可能的命名）
					outputType := "unknown"
					// 驼峰命名
					if _, ok := outputMap["MessageOutput"]; ok {
						outputType = "message"
					} else if _, ok := outputMap["ToolCallOutput"]; ok {
						outputType = "tool_call"
					} else if _, ok := outputMap["ToolResultOutput"]; ok {
						outputType = "tool_result"
					}
					// snake_case 命名（备选）
					if outputType == "unknown" {
						if _, ok := outputMap["message_output"]; ok {
							outputType = "message_snake"
						} else if _, ok := outputMap["tool_call_output"]; ok {
							outputType = "tool_call_snake"
						} else if _, ok := outputMap["tool_result_output"]; ok {
							outputType = "tool_result_snake"
						}
					}
					logger.Info("检测到的输出类型: %s", outputType)

					// 记录 assistant 消息（支持 MessageOutput 和 message_output 两种命名）
					var msgOutput interface{}
					var hasMsgOutput bool
					if msgOutput, hasMsgOutput = outputMap["MessageOutput"]; !hasMsgOutput {
						msgOutput, hasMsgOutput = outputMap["message_output"]
					}
					if hasMsgOutput {
						if msgMap, ok := msgOutput.(map[string]interface{}); ok {
							// 尝试提取消息内容
							// eino 的结构可能是: MessageOutput -> Message -> content
							msgContent := ""

							// 首先尝试访问嵌套的 Message 字段（大写和小写）
							for _, msgKey := range []string{"Message", "message"} {
								if messageObj, ok := msgMap[msgKey]; ok {
									if messageMap, ok := messageObj.(map[string]interface{}); ok {
										for _, contentKey := range []string{"content", "Content", "text", "Text"} {
											if content, ok := messageMap[contentKey]; ok {
												if contentStr, ok := content.(string); ok {
													msgContent = contentStr
													break
												} else {
													msgContent = fmt.Sprintf("%v", content)
													break
												}
											}
										}
									}
									if msgContent != "" {
										break
									}
								}
							}

							// 如果没有嵌套结构，尝试直接访问
							if msgContent == "" {
								for _, contentKey := range []string{"content", "Content", "text", "Text"} {
									if content, ok := msgMap[contentKey]; ok {
										if contentStr, ok := content.(string); ok {
											msgContent = contentStr
											break
										} else {
											msgContent = fmt.Sprintf("%v", content)
											break
										}
									}
								}
							}

							// 如果还是没找到，尝试序列化整个消息
							if msgContent == "" {
								msgBytes, _ := json.Marshal(msgMap)
								msgContent = string(msgBytes)
							}

							if msgContent != "" {
								logEntry := AgentLogEntry{
									Type:      "assistant",
									Content:   msgContent,
									Timestamp: time.Now().Format("2006-01-02 15:04:05"),
								}
								agentLogs = append(agentLogs, logEntry)
								a.logAndEmit(logEntry)
							}
						}
					}

					// 记录工具调用（支持 ToolCallOutput 和 tool_call_output 两种命名）
					var toolCallOutput interface{}
					var hasToolCallOutput bool
					if toolCallOutput, hasToolCallOutput = outputMap["ToolCallOutput"]; !hasToolCallOutput {
						toolCallOutput, hasToolCallOutput = outputMap["tool_call_output"]
					}
					if hasToolCallOutput {
						if toolCallMap, ok := toolCallOutput.(map[string]interface{}); ok {
							toolName := ""
							// 尝试多种可能的字段名
							for _, key := range []string{"ToolName", "tool_name", "Name", "name"} {
								if name, ok := toolCallMap[key].(string); ok && name != "" {
									toolName = name
									break
								}
							}

							// 解析参数
							argsStr := ""
							var args interface{}
							for _, key := range []string{"Arguments", "arguments", "Args", "args"} {
								if a, ok := toolCallMap[key]; ok {
									args = a
									break
								}
							}
							if args != nil {
								// 尝试格式化参数，如果是 zotero_recommend，提供更友好的描述
								if toolName == "zotero_recommend" {
									if argsMap, ok := args.(map[string]interface{}); ok {
										parts := make([]string, 0)
										if platforms, ok := argsMap["platforms"].([]interface{}); ok {
											pStrs := make([]string, 0)
											for _, p := range platforms {
												pStrs = append(pStrs, fmt.Sprintf("%v", p))
											}
											parts = append(parts, fmt.Sprintf("平台: [%s]", strings.Join(pStrs, ", ")))
										}
										if forceCrawl, ok := argsMap["force_crawl"].(bool); ok && forceCrawl {
											parts = append(parts, "强制爬取: 是")
										}
										if maxRec, ok := argsMap["max_recommendations"]; ok {
											parts = append(parts, fmt.Sprintf("最大推荐数: %v", maxRec))
										}
										argsStr = strings.Join(parts, ", ")
									}
								}

								// 如果没有格式化成功，或者不是特定工具，使用默认 JSON
								if argsStr == "" {
									argsBytes, _ := json.Marshal(args)
									argsStr = string(argsBytes)
								}
							}

							content := fmt.Sprintf("调用工具: %s", toolName)
							if argsStr != "" {
								content += fmt.Sprintf("\n参数: %s", argsStr)
							}

							logEntry := AgentLogEntry{
								Type:      "tool_call",
								Content:   content,
								Timestamp: time.Now().Format("2006-01-02 15:04:05"),
							}
							agentLogs = append(agentLogs, logEntry)
							a.logAndEmit(logEntry)
						}
					}

					// 记录工具结果（支持 ToolResultOutput 和 tool_result_output 两种命名）
					var toolResultOutput interface{}
					var hasToolResultOutput bool
					if toolResultOutput, hasToolResultOutput = outputMap["ToolResultOutput"]; !hasToolResultOutput {
						toolResultOutput, hasToolResultOutput = outputMap["tool_result_output"]
					}
					if hasToolResultOutput {
						if toolResultMap, ok := toolResultOutput.(map[string]interface{}); ok {
							toolName := ""
							// 尝试多种可能的字段名
							for _, key := range []string{"ToolName", "tool_name", "Name", "name"} {
								if name, ok := toolResultMap[key].(string); ok && name != "" {
									toolName = name
									break
								}
							}
							// 尝试多种可能的结果字段名
							var result interface{}
							var hasResult bool
							for _, key := range []string{"Result", "result", "Output", "output"} {
								if r, ok := toolResultMap[key]; ok {
									result = r
									hasResult = true
									break
								}
							}
							if hasResult {
								// 如果是 zotero_recommend 工具，尝试解析结果
								if toolName == "zotero_recommend" {
									logger.Info("检测到 zotero_recommend 工具结果，尝试解析")
									if resultMap, ok := result.(map[string]interface{}); ok {
										resultJSON, _ := json.Marshal(resultMap)
										var output ZoteroRecommendOutput
										if err := json.Unmarshal(resultJSON, &output); err != nil {
											logger.Warn("解析 zotero_recommend 结果失败: %v", err)
											logger.Debug("原始结果 JSON: %s", string(resultJSON))
										} else if output.Success {
											logger.Info("成功解析 zotero_recommend 结果，包含 %d 个推荐组，共 %d 篇论文",
												len(output.Recommendations),
												func() int {
													total := 0
													for _, g := range output.Recommendations {
														total += len(g.Papers)
													}
													return total
												}())
											finalOutput = &output
										} else {
											logger.Warn("zotero_recommend 工具返回 Success=false, Message: %s", output.Message)
										}
									} else {
										logger.Warn("zotero_recommend 工具结果不是 map 类型: %T", result)
									}
								}

								// 记录工具结果日志（简化显示，不显示完整 JSON）
								content := fmt.Sprintf("✅ 工具 %s 执行完成", toolName)
								if toolName == "zotero_recommend" && finalOutput != nil {
									total := 0
									for _, g := range finalOutput.Recommendations {
										total += len(g.Papers)
									}
									content += fmt.Sprintf("\n结果: 成功推荐 %d 组，共 %d 篇论文", len(finalOutput.Recommendations), total)
								}

								logEntry := AgentLogEntry{
									Type:      "tool_result",
									Content:   content,
									Timestamp: time.Now().Format("2006-01-02 15:04:05"),
								}
								agentLogs = append(agentLogs, logEntry)
								a.logAndEmit(logEntry)
							}
						}
					}

					// 处理扁平结构的工具调用（直接在顶层有 tool_call_name 等字段）
					if outputType == "unknown" {
						// 检查是否是扁平的工具调用结构
						if toolCallName, ok := outputMap["tool_call_name"].(string); ok {
							outputType = "tool_call_flat"
							content := fmt.Sprintf("调用工具: %s", toolCallName)
							if toolCallID, ok := outputMap["tool_call_id"].(string); ok {
								content += fmt.Sprintf("\nID: %s", toolCallID)
							}
							logEntry := AgentLogEntry{
								Type:      "tool_call",
								Content:   content,
								Timestamp: time.Now().Format("2006-01-02 15:04:05"),
							}
							agentLogs = append(agentLogs, logEntry)
							a.logAndEmit(logEntry)
						}
						// 检查是否是扁平的工具结果结构
						if toolName, ok := outputMap["tool_name"].(string); ok {
							outputType = "tool_result_flat"
							// 尝试获取结果
							if result, ok := outputMap["result"]; ok {
								if toolName == "zotero_recommend" {
									logger.Info("检测到扁平结构的 zotero_recommend 工具结果")
									if resultMap, ok := result.(map[string]interface{}); ok {
										resultJSON, _ := json.Marshal(resultMap)
										var output ZoteroRecommendOutput
										if err := json.Unmarshal(resultJSON, &output); err != nil {
											logger.Warn("解析 zotero_recommend 结果失败: %v", err)
										} else if output.Success {
											logger.Info("成功解析 zotero_recommend 结果（扁平结构），包含 %d 个推荐组", len(output.Recommendations))
											finalOutput = &output
										}
									}
								}
								content := fmt.Sprintf("工具 %s 执行完成", toolName)
								if finalOutput != nil {
									total := 0
									for _, g := range finalOutput.Recommendations {
										total += len(g.Papers)
									}
									content += fmt.Sprintf("\n结果: 成功推荐 %d 组，共 %d 篇论文", len(finalOutput.Recommendations), total)
								}
								logEntry := AgentLogEntry{
									Type:      "tool_result",
									Content:   content,
									Timestamp: time.Now().Format("2006-01-02 15:04:05"),
								}
								agentLogs = append(agentLogs, logEntry)
								a.logAndEmit(logEntry)
							}
						}
					}

					// 如果仍然没有识别到特定类型，记录原始输出（用于调试）
					if outputType == "unknown" && len(outputMap) > 0 {
						logger.Info("未识别的输出类型，记录原始输出")
						debugStr := string(outputJSON)
						if len(debugStr) > 500 {
							debugStr = debugStr[:500] + "..."
						}
						logEntry := AgentLogEntry{
							Type:      "assistant",
							Content:   fmt.Sprintf("输出类型: %s\n内容: %s", outputType, debugStr),
							Timestamp: time.Now().Format("2006-01-02 15:04:05"),
						}
						agentLogs = append(agentLogs, logEntry)
						a.logAndEmit(logEntry)
					}
				}
			}

			// 也尝试使用 MessageOutput API
			if event.Output != nil && event.Output.MessageOutput != nil {
				msg, err := event.Output.MessageOutput.GetMessage()
				if err == nil && msg != nil {
					// 尝试提取消息内容
					msgStr := fmt.Sprintf("%v", msg)
					// 如果还没有记录这条消息，则记录
					if len(agentLogs) == 0 || agentLogs[len(agentLogs)-1].Type != "assistant" || agentLogs[len(agentLogs)-1].Content != msgStr {
						logEntry := AgentLogEntry{
							Type:      "assistant",
							Content:   msgStr,
							Timestamp: time.Now().Format("2006-01-02 15:04:05"),
						}
						agentLogs = append(agentLogs, logEntry)
						a.logAndEmit(logEntry)
					}
				}
			}
		} else {
			logger.Debug("Agent 事件输出为空")
			// 即使输出为空，也记录一条日志
			logEntry := AgentLogEntry{
				Type:      "assistant",
				Content:   fmt.Sprintf("事件 #%d 输出为空", eventCount),
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			}
			agentLogs = append(agentLogs, logEntry)
			a.logAndEmit(logEntry)
		}
	}

	// 确保至少有一条日志（用户查询）
	if len(agentLogs) == 0 {
		logger.Warn("未收集到任何日志，至少应该有一条用户查询日志")
	}

	// 如果没有通过工具获取到结果，回退到直接调用
	if finalOutput == nil {
		logger.Info("Agent 调用未返回结果，回退到直接调用工具逻辑。已收集 %d 条日志，处理了 %d 个事件", len(agentLogs), eventCount)
		// 如果没有收集到任何日志，至少添加用户查询和回退说明
		if len(agentLogs) == 0 {
			logger.Warn("未收集到任何 agent 日志，可能 agent 调用失败或没有产生事件")
			errLog := AgentLogEntry{
				Type:      "assistant",
				Content:   "Agent 调用未产生任何事件，可能 agent 未初始化或调用失败",
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			}
			agentLogs = append(agentLogs, errLog)
			a.logAndEmit(errLog)
		} else {
			// 添加一条日志说明回退
			fallbackLog := AgentLogEntry{
				Type:      "assistant",
				Content:   fmt.Sprintf("Agent 调用未返回结果，使用直接调用方式获取推荐（已处理 %d 个事件）", eventCount),
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			}
			agentLogs = append(agentLogs, fallbackLog)
			a.logAndEmit(fallbackLog)
		}
		// 回退到直接调用工具逻辑
		return a.getDailyRecommendationsDirect(opts, agentLogs)
	}

	logger.Info("Agent 调用成功，收集到 %d 条日志", len(agentLogs))

	// 确保日志不为空
	if len(agentLogs) == 0 {
		logger.Warn("Agent 调用成功但未收集到日志，添加默认日志")
		emptyLog := AgentLogEntry{
			Type:      "assistant",
			Content:   "Agent 调用成功，但未收集到详细日志",
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, emptyLog)
		a.logAndEmit(emptyLog)
	}

	// 转换为前端格式
	recommendResult := RecommendResult{
		CrawledToday:     finalOutput.CrawledToday,
		CrawlCount:       finalOutput.CrawlCount,
		ZoteroPaperCount: finalOutput.ZoteroPaperCount,
		Recommendations:  finalOutput.Recommendations,
		Message:          finalOutput.Message,
		AgentLogs:        agentLogs,
	}

	data, err := json.Marshal(recommendResult)
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}

	logger.Debug("返回的 JSON 数据长度: %d 字节，包含 %d 条日志", len(data), len(agentLogs))

	return string(data), nil
}

// getDailyRecommendationsDirect 直接调用工具逻辑（回退方案）
func (a *App) getDailyRecommendationsDirect(opts RecommendOptions, agentLogs []AgentLogEntry) (string, error) {
	// 转换 RecommendOptions 为内部使用的参数
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
		Success:         true,
		Recommendations: make([]RecommendationGroup, 0),
	}

	// 检查今天是否已爬取（仅当日期范围包含今天时）
	today := time.Now().Format("2006-01-02")
	alreadyCrawled := false
	dateFrom := opts.DateFrom
	if dateFrom == "" {
		dateFrom = today
	}
	dateTo := opts.DateTo
	if dateTo == "" {
		dateTo = today
	}
	if dateFrom <= today && dateTo >= today {
		alreadyCrawled = checkTodayCrawled()
		output.CrawledToday = alreadyCrawled
	}

	// 如果需要爬取（未爬取或强制爬取）
	if !alreadyCrawled || opts.ForceCrawl {
		logger.Info("开始爬取论文（日期范围: %s 至 %s）...", dateFrom, dateTo)
		crawlCount, err := crawlPapers(ctx, a, opts.Platforms, dateFrom, dateTo)
		if err != nil {
			// 爬取失败不影响继续执行
		} else {
			output.CrawlCount = crawlCount
			// 如果日期范围包含今天，标记今天已爬取
			if dateFrom <= today && dateTo >= today {
				if err := markTodayCrawled(); err == nil {
					output.CrawledToday = true
				}
			}
		}
	}

	// 从 Zotero 获取论文
	zoteroPapers, err := getZoteroPapers(opts.ZoteroCollection, 50)
	if err != nil {
		// 记录错误到日志
		errLog := AgentLogEntry{
			Type:      "error",
			Content:   fmt.Sprintf("从 Zotero 获取论文失败: %v", err),
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, errLog)
		a.logAndEmit(errLog)
		// 返回错误结果，但包含日志（不返回 error，而是返回 JSON）
		recommendResult := RecommendResult{
			CrawledToday:     output.CrawledToday,
			CrawlCount:       output.CrawlCount,
			ZoteroPaperCount: 0,
			Recommendations:  make([]RecommendationGroup, 0),
			Message:          fmt.Sprintf("从 Zotero 获取论文失败: %v", err),
			AgentLogs:        agentLogs,
		}
		data, marshalErr := json.Marshal(recommendResult)
		if marshalErr != nil {
			return "", fmt.Errorf("marshal error result failed: %w", marshalErr)
		}
		return string(data), nil
	}

	if len(zoteroPapers) == 0 {
		// 记录警告到日志
		warnLog := AgentLogEntry{
			Type:      "error",
			Content:   "Zotero 中没有找到论文，请先在 Zotero 中添加一些论文",
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, warnLog)
		a.logAndEmit(warnLog)
		// 返回空结果，但包含日志（不返回 error，而是返回 JSON）
		recommendResult := RecommendResult{
			CrawledToday:     output.CrawledToday,
			CrawlCount:       output.CrawlCount,
			ZoteroPaperCount: 0,
			Recommendations:  make([]RecommendationGroup, 0),
			Message:          "Zotero 中没有找到论文，请先在 Zotero 中添加一些论文",
			AgentLogs:        agentLogs,
		}
		data, marshalErr := json.Marshal(recommendResult)
		if marshalErr != nil {
			return "", fmt.Errorf("marshal empty result failed: %w", marshalErr)
		}
		return string(data), nil
	}

	output.ZoteroPaperCount = len(zoteroPapers)

	// 解析日期范围用于搜索
	var fromDate, toDate *time.Time
	if opts.DateFrom != "" && opts.DateTo != "" {
		from, err := time.Parse("2006-01-02", opts.DateFrom)
		if err == nil {
			from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
			fromDate = &from
		}
		to, err := time.Parse("2006-01-02", opts.DateTo)
		if err == nil {
			to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
			toDate = &to
		}
	}

	// 为每篇 Zotero 论文搜索相似的新论文
	allRecommendedPapers := make(map[string]*models.SimilarPaper)

	for _, zoteroPaper := range zoteroPapers {
		similarPapers, err := searchSimilarPapers(ctx, a, zoteroPaper, topK, fromDate, toDate)
		if err != nil {
			continue
		}

		// 过滤重复
		filteredPapers := make([]*models.SimilarPaper, 0)
		for _, sp := range similarPapers {
			key := fmt.Sprintf("%s:%s", sp.Paper.Source, sp.Paper.SourceID)
			if _, exists := allRecommendedPapers[key]; !exists {
				isDuplicate := false
				for _, zp := range zoteroPapers {
					if zp.Source == sp.Paper.Source && zp.SourceID == sp.Paper.SourceID {
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
				ZoteroPaper: *zoteroPaper,
				Papers:      filteredPapers,
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
			Content:   fmt.Sprintf("未找到匹配的推荐论文。已搜索 %d 篇 Zotero 论文，但未找到相似的新论文。", len(zoteroPapers)),
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, notFoundLog)
		a.logAndEmit(notFoundLog)
		output.Message = fmt.Sprintf("未找到匹配的推荐论文，基于 %d 篇 Zotero 论文", len(zoteroPapers))
	} else {
		successLog := AgentLogEntry{
			Type:      "assistant",
			Content:   fmt.Sprintf("成功推荐 %d 篇论文，基于 %d 篇 Zotero 论文", totalRecommended, len(zoteroPapers)),
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		agentLogs = append(agentLogs, successLog)
		a.logAndEmit(successLog)
		output.Message = fmt.Sprintf("成功推荐 %d 篇论文，基于 %d 篇 Zotero 论文", totalRecommended, len(zoteroPapers))
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
		CrawledToday:     output.CrawledToday,
		CrawlCount:       output.CrawlCount,
		ZoteroPaperCount: output.ZoteroPaperCount,
		Recommendations:  output.Recommendations,
		Message:          output.Message,
		AgentLogs:        agentLogs,
	}

	data, err := json.Marshal(recommendResult)
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}

	logger.Debug("返回的 JSON 数据长度: %d 字节", len(data))
	logger.Debug("返回的 JSON 数据预览: %s", string(data)[:min(500, len(data))])

	return string(data), nil
}

// formatToolParams 格式化工具参数为字符串
func formatToolParams(params map[string]interface{}) string {
	parts := make([]string, 0)
	for k, v := range params {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}
