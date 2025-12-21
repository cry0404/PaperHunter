package core

import (
	"context"
	"fmt"

	storage "PaperHunter/db"
	emb "PaperHunter/internal/embedding"
	"PaperHunter/internal/ir"
	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"
)

// Searcher 本地检索器，支持语义搜索、关键词搜索和IR搜索
type Searcher struct {
	db         storage.PaperStorage
	embedder   emb.Service
	irSearcher *ir.IRSearcher // IR搜索引擎
}

func NewSearcher(db storage.PaperStorage, embedder emb.Service) *Searcher {
	// 创建IR搜索引擎的分词器
	tokenizer, err := ir.NewTokenizer()
	if err != nil {
		logger.Warn("创建IR分词器失败: %v", err)
	}

	var irSearcher *ir.IRSearcher
	if tokenizer != nil {
		irSearcher = ir.NewIRSearcher(tokenizer)
	}

	return &Searcher{
		db:         db,
		embedder:   embedder,
		irSearcher: irSearcher,
	}
}

// SearchOptions 搜索参数
type SearchOptions struct {
	// 查询文本（用于语义搜索）
	Query string
	// 示例论文列表（用于基于 example 的搜索）
	Examples []*models.Paper
	// 过滤条件
	Condition models.SearchCondition
	// 返回前 K 个结果
	TopK int
	// 是否使用语义搜索（需要配置 embedder）
	Semantic bool
	// IR搜索模式
	IR          bool   // 是否使用IR搜索
	IRAlgorithm string // IR算法类型: "tfidf", "bm25", "all"
}

// Search 执行搜索
// - IR搜索: 使用TF-IDF或BM25算法进行传统信息检索
// - 语义搜索: 将 query/examples 转为向量，在数据库中查找相似论文
// - 关键词搜索: 在标题和摘要中使用 SQL LIKE 查询
func (s *Searcher) Search(ctx context.Context, opts SearchOptions) ([]*models.SimilarPaper, error) {
	// IR搜索模式
	if opts.IR {
		return s.searchWithIR(ctx, opts)
	}

	// 关键词搜索模式
	if !opts.Semantic {
		if opts.Query == "" {
			return nil, fmt.Errorf("关键词搜索需要提供查询文本(--query)")
		}

		logger.Info("使用关键词搜索: %s", opts.Query)
		papers, err := s.db.SearchByKeywords(opts.Query, opts.Condition)
		if err != nil {
			return nil, fmt.Errorf("关键词搜索失败: %w", err)
		}

		// 将结果转换为 SimilarPaper 格式（相似度设为 1.0）
		results := make([]*models.SimilarPaper, 0, len(papers))
		for _, p := range papers {
			results = append(results, &models.SimilarPaper{
				Paper:      *p,
				Similarity: 1.0,
			})
		}

		// 应用 TopK 限制
		if opts.TopK > 0 && len(results) > opts.TopK {
			results = results[:opts.TopK]
		}

		logger.Info("关键词搜索完成，返回 %d 篇相关论文", len(results))
		return results, nil
	}

	// 语义搜索模式

	if s.embedder == nil {
		return nil, fmt.Errorf("语义搜索需要配置 embedding 服务，请检查配置文件中的 embedding.apikey")
	}

	var queryVec []float32
	var err error

	if len(opts.Examples) > 0 {
		queryVec, err = s.embedFromExamples(ctx, opts.Examples)
	} else if opts.Query != "" {
		logger.Info("使用查询文本进行搜索: %s", opts.Query)
		queryVec, err = s.embedder.EmbedQuery(ctx, opts.Query)
	} else {
		return nil, fmt.Errorf("请提供查询文本(--query)或示例论文(--examples)")
	}

	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	logger.Debug("查询向量维度: %d", len(queryVec))

	results, err := s.db.SearchByEmbedding(queryVec, s.embedder.ModelName(), opts.Condition, opts.TopK)
	if err != nil {
		return nil, fmt.Errorf("数据库检索失败: %w", err)
	}

	return results, nil
}

// embedFromExamples 从多个示例论文生成平均向量
func (s *Searcher) embedFromExamples(ctx context.Context, examples []*models.Paper) ([]float32, error) {
	texts := make([]string, 0, len(examples))
	for _, ex := range examples {
		text := emb.BuildEmbeddingText(ex)
		texts = append(texts, text)
	}

	logger.Debug("正在为 %d 个示例生成向量...", len(texts))
	vecs, err := s.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return nil, err
	}

	if len(vecs) == 0 {
		return nil, fmt.Errorf("示例向量生成失败")
	}

	dim := len(vecs[0])
	avgVec := make([]float32, dim)
	for _, vec := range vecs {
		for i := range vec {
			avgVec[i] += vec[i]
		}
	}
	for i := range avgVec {
		avgVec[i] /= float32(len(vecs))
	}

	logger.Debug("生成平均向量，维度: %d", dim)
	return avgVec, nil
}

// ComputeMissingEmbeddings 批量计算缺失的 embedding
// 用于为已爬取的论文补充向量数据
func (s *Searcher) ComputeMissingEmbeddings(ctx context.Context, batchSize int) (int, error) {
	if s.embedder == nil {
		return 0, fmt.Errorf("未配置 embedding 服务")
	}

	model := s.embedder.ModelName()
	papers, err := s.db.GetPapersNeedingEmbedding(model, batchSize)
	if err != nil {
		return 0, fmt.Errorf("获取待处理论文失败: %w", err)
	}

	if len(papers) == 0 {
		logger.Info("没有需要计算向量的论文")
		return 0, nil
	}

	logger.Info("开始为 %d 篇论文计算向量", len(papers))

	count := 0
	for i, p := range papers {
		text := emb.BuildEmbeddingText(p)
		vec, err := s.embedder.EmbedQuery(ctx, text)
		if err != nil {
			logger.Warn("[%d/%d] 向量生成失败 (paper_id=%d): %v", i+1, len(papers), p.ID, err)
			continue
		}

		if err := s.db.SaveEmbedding(p.ID, model, text, vec); err != nil {
			logger.Warn("[%d/%d] 向量保存失败 (paper_id=%d): %v", i+1, len(papers), p.ID, err)
			continue
		}

		logger.Debug("[%d/%d] 向量保存成功: paper_id=%d, dim=%d", i+1, len(papers), p.ID, len(vec))
		count++
	}

	logger.Info("向量计算完成: %d/%d 成功", count, len(papers))
	return count, nil
}

// searchWithIR 使用传统IR算法进行搜索
func (s *Searcher) searchWithIR(ctx context.Context, opts SearchOptions) ([]*models.SimilarPaper, error) {
	if s.irSearcher == nil {
		return nil, fmt.Errorf("IR搜索引擎未初始化")
	}

	if opts.Query == "" {
		return nil, fmt.Errorf("IR搜索需要提供查询文本")
	}

	// 如果索引为空，需要构建索引
	if s.irSearcher.IsEmpty() {
		logger.Info("IR索引为空，正在从数据库构建索引...")
		papers, err := s.getAllPapersForIR(ctx)
		if err != nil {
			return nil, fmt.Errorf("获取论文数据失败: %w", err)
		}

		if len(papers) == 0 {
			return nil, fmt.Errorf("数据库中没有论文数据")
		}

		err = s.irSearcher.BuildIndex(papers)
		if err != nil {
			return nil, fmt.Errorf("构建IR索引失败: %w", err)
		}

		logger.Info("IR索引构建完成，包含 %d 篇论文", len(papers))
	}

	// 设置默认值
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	if opts.IRAlgorithm == "" {
		opts.IRAlgorithm = "bm25"
	}

	// 执行搜索
	var irResults []*ir.SearchResult
	var err error

	if opts.IRAlgorithm == "all" {
		// 执行多算法搜索，用于对比
		results, err := s.irSearcher.SearchMultiple(opts.Query, opts.TopK, []string{"tfidf", "bm25"})
		if err != nil {
			return nil, fmt.Errorf("IR搜索失败: %w", err)
		}

		// 使用BM25结果作为主要结果（实际应用中可以选择最好的结果）
		if bm25Results, exists := results["bm25"]; exists {
			irResults = bm25Results
		} else {
			irResults = make([]*ir.SearchResult, 0)
		}
	} else {
		// 执行单一算法搜索
		irOpts := ir.SearchOptions{
			Query:     opts.Query,
			TopK:      opts.TopK,
			Algorithm: opts.IRAlgorithm,
		}

		irResults, err = s.irSearcher.Search(irOpts)
		if err != nil {
			return nil, fmt.Errorf("IR搜索失败: %w", err)
		}
	}

	// 转换结果格式
	similarPapers := make([]*models.SimilarPaper, 0, len(irResults))
	for _, result := range irResults {
		if result.Paper != nil {
			similarPaper := &models.SimilarPaper{
				Paper:      *result.Paper,
				Similarity: float32(result.Score), // BM25/TF-IDF分数作为相似度
			}
			similarPapers = append(similarPapers, similarPaper)
		}
	}

	logger.Info("IR搜索(%s)完成，返回 %d 篇相关论文", opts.IRAlgorithm, len(similarPapers))
	return similarPapers, nil
}

// getAllPapersForIR 获取所有论文用于构建IR索引
func (s *Searcher) getAllPapersForIR(ctx context.Context) ([]*models.Paper, error) {
	// 设置一个较大的limit来获取所有论文
	largeLimit := 10000

	papers, err := s.db.GetPapersByConditions([]string{}, []interface{}{}, largeLimit)
	if err != nil {
		return nil, fmt.Errorf("从数据库获取论文失败: %w", err)
	}

	// GetPapersByConditions 已经返回了 []*models.Paper，所以直接使用
	return papers, nil
}

// GetIRStats 获取IR搜索引擎的统计信息
func (s *Searcher) GetIRStats() map[string]interface{} {
	if s.irSearcher == nil {
		return map[string]interface{}{
			"initialized": false,
			"message":     "IR搜索引擎未初始化",
		}
	}

	stats := s.irSearcher.GetIndexStats()
	stats["initialized"] = true
	return stats
}

// SetBM25Parameters 设置BM25参数
func (s *Searcher) SetBM25Parameters(k1, b float64) error {
	if s.irSearcher == nil {
		return fmt.Errorf("IR搜索引擎未初始化")
	}

	s.irSearcher.SetBM25Parameters(k1, b)
	logger.Info("BM25参数已更新: k1=%.2f, b=%.2f", k1, b)
	return nil
}

// AddPaperToIR 添加论文到 IR 索引
func (s *Searcher) AddPaperToIR(paper *models.Paper) {
	if s.irSearcher != nil {
		if err := s.irSearcher.AddDocument(paper); err != nil {
			logger.Warn("添加论文到IR索引失败: %v", err)
		}
	}
}
