package core

import (
	"context"
	"fmt"

	storage "PaperHunter/db"
	emb "PaperHunter/internal/embedding"
	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"
)

// Searcher 本地检索器，支持语义搜索和关键词搜索
type Searcher struct {
	db       storage.PaperStorage
	embedder emb.Service
}

func NewSearcher(db storage.PaperStorage, embedder emb.Service) *Searcher {
	return &Searcher{
		db:       db,
		embedder: embedder,
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
}

// Search 执行搜索
// - 语义搜索: 将 query/examples 转为向量，在数据库中查找相似论文
// - 关键词搜索: 在标题和摘要中使用 SQL LIKE 查询
func (s *Searcher) Search(ctx context.Context, opts SearchOptions) ([]*models.SimilarPaper, error) {
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

	logger.Info("检索完成，返回 %d 篇相关论文", len(results))
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
