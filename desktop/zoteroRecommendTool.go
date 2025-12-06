package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"PaperHunter/config"
	"PaperHunter/internal/core"
	"PaperHunter/internal/models"
	"PaperHunter/internal/platform/arxiv"
	"PaperHunter/pkg/logger"
	"PaperHunter/pkg/upload/zotero"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ZoteroRecommendInput 统一的 Zotero 和推荐工具输入参数（专注于arXiv每日推荐）

type ZoteroRecommendInput struct {
	// Action 操作类型：get_collections（获取集合列表）、get_papers（获取论文）、daily_recommend（arXiv每日推荐）
	Action string `json:"action" jsonschema:"required,enum=get_collections,enum=get_papers,enum=daily_recommend,description=Action to perform: get_collections, get_papers, or arXiv daily_recommend"`


	CollectionKey string `json:"collection_key,omitempty" jsonschema:"description=Collection key for get_papers or daily_recommend action"`
	Limit         int    `json:"limit,omitempty" jsonschema:"description=Limit number of papers to return (for get_papers)"`


	TopK               int    `json:"top_k,omitempty" jsonschema:"description=Number of recommended papers (default: 10)"`
	MaxRecommendations int    `json:"max_recommendations,omitempty" jsonschema:"description=Maximum total number of papers to recommend (default: 30)"`
	ForceCrawl         bool   `json:"force_crawl,omitempty" jsonschema:"description=Force re-crawl today's arXiv papers (default: false)"`
	DateFrom           string `json:"date_from,omitempty" jsonschema:"description=Date in YYYY-MM-DD format (default: today)"`
	DateTo             string `json:"date_to,omitempty" jsonschema:"description=Date in YYYY-MM-DD format (default: today)"`
	ExampleTitle       string `json:"example_title,omitempty" jsonschema:"description=Your research interests or topic (used for recommendation)"`
	ExampleAbstract    string `json:"example_abstract,omitempty" jsonschema:"description=Detailed description of your research interests"`

	// 新增：本地JSON文件导入支持
	LocalFilePath   string `json:"local_file_path,omitempty" jsonschema:"description=Path to local JSON file to import for recommendation"`
	LocalFileAction string `json:"local_file_action,omitempty" jsonschema:"description=Action: 'import_for_recommend'"`
}


type ZoteroRecommendOutput struct {
	Success bool `json:"success" jsonschema:"description=Whether the operation was successful"`


	Message string `json:"message,omitempty" jsonschema:"description=Result message"`
	Data    any    `json:"data,omitempty" jsonschema:"description=Result data (collections or papers for get_collections/get_papers)"`

	// 用于 daily_recommend（arXiv专注）
	CrawledToday    bool                  `json:"crawled_today,omitempty" jsonschema:"description=Whether arXiv papers were crawled today (for daily_recommend)"`
	ArxivCrawlCount int                   `json:"arxiv_crawl_count,omitempty" jsonschema:"description=Number of arXiv papers crawled today (for daily_recommend)"`
	SeedPaperCount  int                   `json:"seed_paper_count,omitempty" jsonschema:"description=Number of seed papers used for recommendation (Zotero + interests)"`
	Recommendations []RecommendationGroup `json:"recommendations,omitempty" jsonschema:"description=Grouped recommendations based on seed papers or interests (for daily_recommend)"`
}


type RecommendationGroup struct {
	SeedPaper models.Paper           `json:"seed_paper" jsonschema:"description=The seed paper or interest this group is based on (Zotero paper or user interest)"`
	Papers    []*models.SimilarPaper `json:"papers" jsonschema:"description=Recommended arXiv papers similar to the seed paper or interest"`
}


func getTodayCrawlStatusFile() string {
	homeDir, _ := os.UserHomeDir()
	statusDir := filepath.Join(homeDir, ".quicksearch", "status")
	os.MkdirAll(statusDir, 0755)
	today := time.Now().Format("2006-01-02")
	return filepath.Join(statusDir, fmt.Sprintf("crawl_%s.txt", today))
}


func checkTodayCrawled() bool {
	statusFile := getTodayCrawlStatusFile()
	_, err := os.Stat(statusFile)
	return err == nil
}


func markTodayCrawled() error {
	statusFile := getTodayCrawlStatusFile()
	return os.WriteFile(statusFile, []byte(time.Now().Format(time.RFC3339)), 0644)
}


// crawlTodayNewSubmissions 爬取今日 arXiv New Submissions 页面的论文
// 使用 https://arxiv.org/list/cs/new 获取今日公布的 CS 领域论文
func crawlTodayNewSubmissions(ctx context.Context, app *App, category string) (int, error) {
	if app == nil || app.coreApp == nil {
		return 0, fmt.Errorf("app instance is not initialized")
	}

	if category == "" {
		category = "cs" // 默认 CS 全部
	}

	// 检查是否为周末（arXiv 在周末通常不发刊）


	logger.Info("使用 New Submissions 页面获取今日 arXiv %s 论文", category)

	// 获取 arxiv adapter
	plat, err := app.coreApp.GetPlatform("arxiv")
	if err != nil {
		return 0, fmt.Errorf("获取 arxiv 平台失败: %w", err)
	}

	arxivAdapter, ok := plat.(*arxiv.Adapter)
	if !ok {
		return 0, fmt.Errorf("类型转换失败: 不是 arxiv.Adapter")
	}

	// 获取今日新论文
	result, err := arxivAdapter.FetchNewSubmissions(ctx, category)
	if err != nil {
		return 0, fmt.Errorf("获取今日新论文失败: %w", err)
	}

	if len(result.Papers) == 0 {
		logger.Info("今日没有新论文")
		return 0, nil
	}

	logger.Info("获取到 %d 篇今日新论文，开始保存到数据库", len(result.Papers))

	// 保存到数据库
	count, err := app.coreApp.SavePapers(ctx, result.Papers)
	if err != nil {
		logger.Warn("保存论文时出错: %v", err)
	}

	logger.Info("今日 arXiv %s 新论文保存完成: %d 篇", category, count)
	return count, nil
}

// getZoteroPapers 从 Zotero 获取论文
func getZoteroPapers(collectionKey string, limit int) ([]*models.Paper, error) {
	cfg := config.Get()
	if cfg.Zotero.UserID == "" || cfg.Zotero.APIKey == "" {
		return nil, fmt.Errorf("zotero 配置不完整，请在配置文件中设置 zotero.user_id 和 zotero.api_key")
	}

	client := zotero.NewClient(cfg.Zotero.UserID, cfg.Zotero.APIKey)
	papers, err := client.GetPapers(collectionKey, limit)
	if err != nil {
		return nil, fmt.Errorf("从 Zotero 获取论文失败: %w", err)
	}

	return papers, nil
}

// searchSimilarPapers 基于种子论文搜索相似论文，支持日期过滤
func searchSimilarPapers(ctx context.Context, app *App, seedPaper *models.Paper, topK int, fromDate, toDate *time.Time) ([]*models.SimilarPaper, error) {
	if app == nil || app.coreApp == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	// 构建搜索条件
	cond := models.SearchCondition{
		Limit:    topK * 3, // 多搜索一些，后续可以过滤
		Sources:  []string{"arxiv"},
		DateFrom: fromDate,
		DateTo:   toDate,
	}

	// 使用语义搜索
	opts := core.SearchOptions{
		Examples:  []*models.Paper{seedPaper},
		Condition: cond,
		TopK:      topK * 3,
		Semantic:  true,
	}

	results, err := app.coreApp.Search(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 过滤低相似度结果（降低阈值以获得更多结果）
	const minSimilarity = 0.2 // 降低阈值
	filtered := make([]*models.SimilarPaper, 0, len(results))
	for _, sp := range results {
		if sp.Similarity >= minSimilarity {
			filtered = append(filtered, sp)
		}
	}

	// 限制返回数量
	if len(filtered) > topK {
		filtered = filtered[:topK]
	}

	logger.Info("搜索完成: 原始 %d 篇，过滤后 %d 篇 (阈值: %.2f)", len(results), len(filtered), minSimilarity)
	return filtered, nil
}

// NewZoteroRecommendTool 创建统一的 Zotero 和推荐工具，接受 App 实例
func NewZoteroRecommendTool(app *App) tool.InvokableTool {
	tool, err := utils.InferTool("zotero_recommend",
		"Simple arXiv recommendations with JSON file import. Actions: get_collections (Zotero), get_papers (Zotero papers), daily_recommend (arXiv CS recommendations). For daily_recommend: either (1) describe research interests in example_title/abstract to get today's arXiv CS paper recommendations, or (2) provide local_file_path to JSON file for import-based recommendations. Set local_file_action to 'import_for_recommend' to use the file as recommendation seed. JSON file format: {\"title\": \"...\", \"abstract\": \"...\"}.",
		func(ctx context.Context, input *ZoteroRecommendInput) (output *ZoteroRecommendOutput, err error) {
			cfg := config.Get()
			if cfg.Zotero.UserID == "" || cfg.Zotero.APIKey == "" {
				return &ZoteroRecommendOutput{
					Success: false,
					Message: "Zotero 配置不完整，请在配置文件中设置 zotero.user_id 和 zotero.api_key",
				}, fmt.Errorf("zotero config incomplete")
			}

			client := zotero.NewClient(cfg.Zotero.UserID, cfg.Zotero.APIKey)

			switch input.Action {
			case "get_collections":
				collections, err := client.GetCollections()
				if err != nil {
					return &ZoteroRecommendOutput{
						Success: false,
						Message: fmt.Sprintf("获取集合列表失败: %v", err),
					}, err
				}

				return &ZoteroRecommendOutput{
					Success: true,
					Message: fmt.Sprintf("成功获取 %d 个集合", len(collections)),
					Data:    collections,
				}, nil

			case "get_papers":
				limit := input.Limit
				if limit <= 0 {
					limit = 100 // 默认限制
				}

				papers, err := client.GetPapers(input.CollectionKey, limit)
				if err != nil {
					errMsg := err.Error()
					if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") {

						if input.CollectionKey != "" {
							logger.Warn("指定的 collection 不存在，尝试获取所有论文")
							papers, err = client.GetPapers("", limit)
							if err != nil {
								return &ZoteroRecommendOutput{
									Success: false,
									Message: fmt.Sprintf("获取论文失败: %v", err),
								}, err
							}
							return &ZoteroRecommendOutput{
								Success: true,
								Message: fmt.Sprintf("指定的 collection 不存在，已获取所有论文 %d 篇", len(papers)),
								Data:    papers,
							}, nil
						}
					}
					return &ZoteroRecommendOutput{
						Success: false,
						Message: fmt.Sprintf("获取论文失败: %v", err),
					}, err
				}

				return &ZoteroRecommendOutput{
					Success: true,
					Message: fmt.Sprintf("成功获取 %d 篇论文", len(papers)),
					Data:    papers,
				}, nil

			case "daily_recommend":
				if app == nil || app.coreApp == nil {
					return &ZoteroRecommendOutput{
						Success: false,
						Message: "app instance is not initialized",
					}, fmt.Errorf("app instance is not initialized")
				}

				// 设置默认值
				topK := input.TopK
				if topK <= 0 {
					topK = 5
				}
				maxRecommendations := input.MaxRecommendations
				if maxRecommendations <= 0 {
					maxRecommendations = 20
				}

				// 解析日期范围，如果没有指定则使用今天
				var dateFrom, dateTo string
				if input.DateFrom != "" {
					dateFrom = input.DateFrom
				} else {
					dateFrom = time.Now().Format("2006-01-02")
				}
				if input.DateTo != "" {
					dateTo = input.DateTo
				} else {
					dateTo = time.Now().Format("2006-01-02")
				}

				output := &ZoteroRecommendOutput{
					Success:         true,
					Recommendations: make([]RecommendationGroup, 0),
				}

				// 检查今天是否已爬取
				today := time.Now().Format("2006-01-02")
				alreadyCrawled := checkTodayCrawled()
				output.CrawledToday = alreadyCrawled

				// 使用 New Submissions 页面爬取今日论文
				if !alreadyCrawled || input.ForceCrawl {
					logger.Info("使用 New Submissions 页面爬取今日 arXiv CS 论文")
					crawlCount, err := crawlTodayNewSubmissions(ctx, app, "cs")
					if err != nil {
						logger.Warn("爬取失败: %v", err)
					} else {
						output.ArxivCrawlCount = crawlCount
						if crawlCount > 0 {
							markTodayCrawled()
							output.CrawledToday = true
						}
						logger.Info("今日 arXiv CS 论文爬取完成: %d 篇", crawlCount)
					}
				} else {
					logger.Info("今日 arXiv 论文已爬取，跳过")
				}

				// 简化种子收集：优先使用用户兴趣描述
				var seeds []*models.Paper

				// 优先使用用户兴趣描述
				if input.ExampleTitle != "" && input.ExampleAbstract != "" {
					logger.Info("使用研究兴趣: %s", input.ExampleTitle)
					seeds = append(seeds, &models.Paper{
						Title:    input.ExampleTitle,
						Abstract: input.ExampleAbstract,
						Source:   "user_interest",
						SourceID: "interest_seed",
					})
				}


				if input.LocalFilePath != "" && input.LocalFileAction == "import_for_recommend" {
					logger.Info("导入本地JSON文件: %s", input.LocalFilePath)
					localPaper, err := importJSONFile(input.LocalFilePath)
					if err != nil {
						logger.Warn("导入JSON文件失败: %v", err)
					} else {
						seeds = append(seeds, localPaper)
						logger.Info("成功导入JSON论文: %s", localPaper.Title)
					}
				}

				if cfg.Zotero.UserID != "" && cfg.Zotero.APIKey != "" && len(seeds) == 0 {
					zoteroPapers, err := getZoteroPapers(input.CollectionKey, 10)
					if err == nil && len(zoteroPapers) > 0 {
						seeds = append(seeds, zoteroPapers...)
						logger.Info("补充 %d 篇Zotero论文", len(zoteroPapers))
					}
				}

				if len(seeds) == 0 {
					return &ZoteroRecommendOutput{
						Success: false,
						Message: "请描述研究兴趣或提供本地文件以获取推荐",
					}, nil
				}

				output.SeedPaperCount = len(seeds)

				// 解析日期范围用于搜索
				fromDate, err := time.Parse("2006-01-02", dateFrom)
				if err != nil {
					return &ZoteroRecommendOutput{
						Success: false,
						Message: fmt.Sprintf("无效的日期格式: %v", err),
					}, err
				}
				toDate, err := time.Parse("2006-01-02", dateTo)
				if err != nil {
					return &ZoteroRecommendOutput{
						Success: false,
						Message: fmt.Sprintf("无效的日期格式: %v", err),
					}, err
				}
				fromDate = time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, fromDate.Location())
				toDate = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())


				allRecommendedPapers := make(map[string]*models.SimilarPaper)

				for _, seedPaper := range seeds {
					logger.Info("基于种子论文搜索: %s", seedPaper.Title)

					// 简化：使用固定的搜索数量
					similarPapers, err := searchSimilarPapers(ctx, app, seedPaper, topK, &fromDate, &toDate)
					if err != nil {
						logger.Warn("搜索失败: %v", err)
						continue
					}

					// 过滤掉已经在种子列表中的论文（如果是 Zotero 论文）
					filteredPapers := make([]*models.SimilarPaper, 0)
					for _, sp := range similarPapers {
						key := fmt.Sprintf("%s:%s", sp.Paper.Source, sp.Paper.SourceID)
						if _, exists := allRecommendedPapers[key]; !exists {
							// 检查是否与种子论文重复（防止推荐自己）
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

					// 如果已达到最大推荐数量，停止
					if len(allRecommendedPapers) >= maxRecommendations {
						break
					}
				}


				if len(allRecommendedPapers) > maxRecommendations {
					// 这里简化处理，只保留前几个组的推荐
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

				if dateFrom == dateTo && dateFrom == today {
					output.Message = fmt.Sprintf("成功从今日arXiv论文中推荐 %d 篇，基于 %d 个兴趣源", totalRecommended, len(output.Recommendations))
					logger.Info("成功从今日arXiv论文中推荐 %d 篇，基于 %d 个兴趣源", totalRecommended, len(output.Recommendations))
				} else {
					output.Message = fmt.Sprintf("成功推荐 %d 篇arXiv论文，基于 %d 篇种子论文", totalRecommended, len(output.Recommendations))
					logger.Info("成功推荐 %d 篇arXiv论文，基于 %d 篇种子论文", totalRecommended, len(output.Recommendations))
				}

				return output, nil

			default:
				return &ZoteroRecommendOutput{
					Success: false,
					Message: fmt.Sprintf("不支持的操作: %s", input.Action),
				}, fmt.Errorf("unsupported action: %s", input.Action)
			}
		})

	if err != nil {
		log.Fatalf("failed to create zotero recommend tool: %v", err)
	}

	return tool
}

func importJSONFile(filePath string) (*models.Paper, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取JSON文件失败: %w", err)
	}

	var paperData struct {
		Title    string `json:"title"`
		Abstract string `json:"abstract"`
	}

	if err := json.Unmarshal(data, &paperData); err != nil {
		return nil, fmt.Errorf("解析JSON文件失败: %w", err)
	}

	if paperData.Title == "" {
		return nil, fmt.Errorf("JSON文件必须包含title字段")
	}

	if paperData.Abstract == "" {
		return nil, fmt.Errorf("JSON文件必须包含abstract字段")
	}

	paper := &models.Paper{
		Title:    paperData.Title,
		Abstract: paperData.Abstract,
		Authors:  []string{},
		URL:      "",
		Source:   "local_json",
		SourceID: "json_" + filepath.Base(filePath),
	}

	logger.Info("从JSON文件导入论文: %s", paperData.Title)
	return paper, nil
}
