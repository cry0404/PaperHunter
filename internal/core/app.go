package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	storage "PaperHunter/db"
	dbsqlite "PaperHunter/db/sqlite"

	exporter "PaperHunter/internal/core/export"
	csv "PaperHunter/internal/core/export/csv"
	json "PaperHunter/internal/core/export/json"
	emb "PaperHunter/internal/embedding"
	"PaperHunter/internal/models"
	"PaperHunter/internal/platform"
	"PaperHunter/pkg/logger"
	feishu "PaperHunter/pkg/upload/feishu"
	zotero "PaperHunter/pkg/upload/zotero"
)

type ZoteroConfig struct {
	UserID      string `mapstructure:"user_id" yaml:"user_id"`
	APIKey      string `mapstructure:"api_key" yaml:"api_key"`
	LibraryType string `mapstructure:"library_type" yaml:"library_type"`
}

type FeiShuConfig struct {
	AppID     string `mapstructure:"app_id" yaml:"app_id"`
	AppSecret string `mapstructure:"app_secret" yaml:"app_secret"`
}

var GlobalApp *App

type App struct {
	db          storage.PaperStorage
	embedder    emb.Service
	platformCfg map[string]platform.Config
	searcher    *Searcher
	zoteroCfg   ZoteroConfig //上传这部分就不考虑单例模式了？ 不是配置必选项，要使用时再说
	feishuCfg   FeiShuConfig
}

func NewApp(databasePath string, embCfg emb.EmbedderConfig, pCfg map[string]platform.Config, zoteroCfg ZoteroConfig, feishuCfg FeiShuConfig) (*App, error) {
	if databasePath == "" {
		homeDir, _ := os.UserHomeDir()

		databasePath = filepath.Join(homeDir, ".quicksearch", "data")
	}
	sqliteDB, err := dbsqlite.NewSQLiteDB(databasePath)
	if err != nil {
		return nil, err
	}

	embedSvc, err := emb.New(embCfg)
	if err != nil {
		return nil, err
	}
	if pCfg == nil {
		pCfg = map[string]platform.Config{}
	}

	searcher := NewSearcher(sqliteDB, embedSvc)

	app := &App{
		db:          sqliteDB,
		embedder:    embedSvc,
		platformCfg: pCfg,
		searcher:    searcher,
		zoteroCfg:   zoteroCfg,
		feishuCfg:   feishuCfg,
	}

	// 设置全局实例
	GlobalApp = app

	return app, nil
}

func (a *App) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	return a.db.Close()
}

type CrawlProgress func(index int, total int, p *models.Paper, paperID int64)

func (a *App) Crawl(ctx context.Context, platformName string, q platform.Query) (int, error) {
	return a.CrawlWithProgress(ctx, platformName, q, nil)
}

func (a *App) CrawlWithProgress(ctx context.Context, platformName string, q platform.Query, progress CrawlProgress) (int, error) {
	logger.Info("开始爬取平台: %s", platformName)
	prov, ok := Get(platformName)
	if !ok {
		return 0, fmt.Errorf("未知或未实现的平台: %s", platformName)
	}

	pcfg, ok := a.platformCfg[platformName]
	if !ok {
		logger.Debug("使用平台默认配置: %s", platformName)
		pcfg = prov.DefaultConfig()
	}

	logger.Debug("创建平台实例: %s", platformName)
	plat, err := prov.New(pcfg)
	if err != nil {
		logger.Error("创建平台实例失败: %v", err)
		return 0, fmt.Errorf("创建平台实例失败: %w", err)
	}

	logger.Debug("执行搜索查询: keywords=%v, categories=%v, limit=%d", q.Keywords, q.Categories, q.Limit)
	res, err := plat.Search(ctx, q)
	if err != nil {
		logger.Error("平台搜索失败: %v", err)
		return 0, fmt.Errorf("爬取失败: %w", err)
	}
	logger.Info("搜索返回 %d 篇论文", len(res.Papers))
	count := 0
	total := len(res.Papers)
	for i, p := range res.Papers {
		if p == nil {
			continue
		}
		logger.Debug("[%d/%d] 保存论文: %s", i+1, len(res.Papers), p.Title)
		pid, err := a.db.Upsert(p)
		if err != nil {
			logger.Error("保存论文失败 [%s]: %v", p.URL, err)
			return count, fmt.Errorf("保存论文失败(%s): %w", p.URL, err)
		}
		count++

		if progress != nil {
			progress(i, total, p, pid)
		}

		if a.embedder != nil {
			logger.Debug("生成向量: paper_id=%d, model=%s", pid, a.embedder.ModelName())
			text := emb.BuildEmbeddingText(p)
			vec, err := a.embedder.EmbedQuery(ctx, text)
			if err != nil {
				logger.Warn("向量生成失败 [paper_id=%d]: %v", pid, err)
			} else if len(vec) > 0 {
				if err := a.db.SaveEmbedding(pid, a.embedder.ModelName(), text, vec); err != nil {
					logger.Warn("向量保存失败 [paper_id=%d]: %v", pid, err)
				} else {
					logger.Debug("向量保存成功: paper_id=%d, dim=%d", pid, len(vec))
				}
			}
		}
	}
	logger.Info("爬取完成，共保存 %d 篇论文", count)
	return count, nil
}

func (a *App) Search(ctx context.Context, opts SearchOptions) ([]*models.SimilarPaper, error) {
	logger.Info("开始本地搜索")
	return a.searcher.Search(ctx, opts)
}

func (a *App) ComputeMissingEmbeddings(ctx context.Context, batchSize int) (int, error) {
	logger.Info("开始计算缺失的向量")
	return a.searcher.ComputeMissingEmbeddings(ctx, batchSize)
}

func (a *App) CountPapers(ctx context.Context, conditions []string, params []interface{}) (int, error) {
	logger.Info("统计论文数量")
	return a.db.CountPapers(conditions, params)
}

func (a *App) DeletePapers(ctx context.Context, conditions []string, params []interface{}) (int, error) {
	logger.Info("删除论文")
	return a.db.DeletePapers(conditions, params)
}

// GetPapersByPairs 按 source+id 组合批量查询论文（不分页，limit=0 表示全部）
func (a *App) GetPapersByPairs(ctx context.Context, pairs map[string][]string) ([]*models.Paper, error) {
	if len(pairs) == 0 {
		return []*models.Paper{}, nil
	}

	var conditionParts []string
	var params []interface{}

	for source, ids := range pairs {
		if len(ids) == 0 {
			continue
		}
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
		return []*models.Paper{}, nil
	}

	conditions := []string{fmt.Sprintf("(%s)", strings.Join(conditionParts, " OR "))}

	return a.db.GetPapersByConditions(conditions, params, 0)
}

func (a *App) GetPapers(ctx context.Context, page, pageSize int, conditions []string, params []interface{}, orderBy string) ([]*models.Paper, int, error) {
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	return a.db.GetPapersList(pageSize, offset, conditions, params, orderBy)
}

// ExportPapers 导出论文到文件
func (a *App) ExportPapers(ctx context.Context, format string, outputPath string, conditions []string, params []interface{}, limit int) error {
	logger.Info("开始导出论文: 格式=%s, 输出=%s", format, outputPath)

	papers, err := a.db.GetPapersByConditions(conditions, params, limit)
	if err != nil {
		return fmt.Errorf("查询论文失败: %w", err)
	}

	if len(papers) == 0 {
		return fmt.Errorf("没有找到符合条件的论文")
	}

	logger.Info("找到 %d 篇论文待导出", len(papers))

	var exp exporter.Exporter
	switch format {
	case "csv":
		exp = csv.NewCSVExporter()
	case "json":
		exp = json.NewJSONExporter()
	default:
		return fmt.Errorf("不支持的导出格式: %s", format)
	}

	if err := exp.Export(papers, outputPath); err != nil {
		return fmt.Errorf("导出失败: %w", err)
	}

	logger.Info("导出成功: %d 篇论文 -> %s", len(papers), outputPath)
	return nil
}

func (a *App) ExportToZotero(ctx context.Context, collectionKey string, conditions []string, params []interface{}, limit int) error {
	logger.Info("开始导出到 Zotero")

	if a.zoteroCfg.UserID == "" || a.zoteroCfg.APIKey == "" {
		return fmt.Errorf("zotero 配置不完整，请在配置文件中设置 zotero.user_id 和 zotero.api_key")
	}

	papers, err := a.db.GetPapersByConditions(conditions, params, limit)
	if err != nil {
		return fmt.Errorf("查询论文失败: %w", err)
	}

	if len(papers) == 0 {
		return fmt.Errorf("没有找到符合条件的论文")
	}

	logger.Info("找到 %d 篇论文待导出", len(papers))

	client := zotero.NewClient(a.zoteroCfg.UserID, a.zoteroCfg.APIKey)

	if err := client.AddPapers(papers, collectionKey); err != nil {
		return fmt.Errorf("添加到 Zotero 失败: %w", err)
	}

	logger.Info("导出到 Zotero 成功: %d 篇论文", len(papers))
	return nil
}

func (a *App) ExportToFeiShuBitable(ctx context.Context, fileName, folderName string, conditions []string, params []interface{}, limit int) error {
	logger.Info("开始导出到 FeiShu")

	if a.feishuCfg.AppID == "" || a.feishuCfg.AppSecret == "" {
		return fmt.Errorf("feishu 配置不完整，请在配置文件中设置 feishu.app_id 和 feishu.app_secret")
	}

	papers, err := a.db.GetPapersByConditions(conditions, params, limit)
	if err != nil {
		return fmt.Errorf("查询论文失败: %w", err)
	}

	if len(papers) == 0 {
		return fmt.Errorf("没有找到符合条件的论文")
	}

	logger.Info("找到 %d 篇论文待导出", len(papers))

	tmpFile, err := os.CreateTemp("", "quicksearch_*.csv")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	exp := csv.NewCSVExporter()
	if err := exp.Export(papers, tmpPath); err != nil {
		return fmt.Errorf("导出 CSV 失败: %w", err)
	}

	logger.Info("已导出为临时 CSV 文件: %s", tmpPath)

	client := feishu.NewClient(a.feishuCfg.AppID, a.feishuCfg.AppSecret, fileName, folderName)

	if _, err := client.UploadCSVToBitable(tmpPath); err != nil {
		return fmt.Errorf("上传到飞书失败: %w", err)
	}

	logger.Info("导出到飞书成功: %d 篇论文", len(papers))
	return nil
}

func (a *App) ExportToFeiShuBitableWithURL(ctx context.Context, fileName, folderName string, conditions []string, params []interface{}, limit int) (string, error) {
	logger.Info("开始导出到 FeiShu (with URL)")

	if a.feishuCfg.AppID == "" || a.feishuCfg.AppSecret == "" {
		return "", fmt.Errorf("feishu 配置不完整，请在配置文件中设置 feishu.app_id 和 feishu.app_secret")
	}

	papers, err := a.db.GetPapersByConditions(conditions, params, limit)
	if err != nil {
		return "", fmt.Errorf("查询论文失败: %w", err)
	}
	if len(papers) == 0 {
		return "", fmt.Errorf("没有找到符合条件的论文")
	}

	tmpFile, err := os.CreateTemp("", "quicksearch_*.csv")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { tmpFile.Close(); os.Remove(tmpPath) }()

	exp := csv.NewCSVExporter()
	if err := exp.Export(papers, tmpPath); err != nil {
		return "", fmt.Errorf("导出 CSV 失败: %w", err)
	}

	client := feishu.NewClient(a.feishuCfg.AppID, a.feishuCfg.AppSecret, fileName, folderName)
	url, err := client.UploadCSVToBitable(tmpPath)
	if err != nil {
		return "", fmt.Errorf("上传到飞书失败: %w", err)
	}

	logger.Info("导出到飞书成功: %d 篇论文, url=%s", len(papers), url)
	return url, nil
}

func (a *App) ZoteroCfg() ZoteroConfig {
	return a.zoteroCfg
}

func (a *App) GetPlatform(platformName string) (platform.Platform, error) {
	prov, ok := Get(platformName)
	if !ok {
		return nil, fmt.Errorf("未知或未实现的平台: %s", platformName)
	}

	pcfg, ok := a.platformCfg[platformName]
	if !ok {
		pcfg = prov.DefaultConfig()
	}

	return prov.New(pcfg)
}

func (a *App) SavePapers(ctx context.Context, papers []*models.Paper) (int, error) {
	count := 0
	for _, p := range papers {
		if p == nil {
			continue
		}
		pid, err := a.db.Upsert(p)
		if err != nil {
			logger.Error("保存论文失败 [%s]: %v", p.URL, err)
			continue
		}
		count++

		if a.embedder != nil {
			text := emb.BuildEmbeddingText(p)
			vec, err := a.embedder.EmbedQuery(ctx, text)
			if err != nil {
				logger.Warn("向量生成失败 [paper_id=%d]: %v", pid, err)
			} else if len(vec) > 0 {
				if err := a.db.SaveEmbedding(pid, a.embedder.ModelName(), text, vec); err != nil {
					logger.Warn("向量保存失败 [paper_id=%d]: %v", pid, err)
				}
			}
		}
	}
	return count, nil
}

func (a *App) FeishuCfg() FeiShuConfig {
	return a.feishuCfg
}
