package ir

import (
	"math"
	"PaperHunter/internal/models"
	"sort"
)

// BM25Searcher BM25 搜索器
type BM25Searcher struct {
	index     *InvertedIndex
	tokenizer *Tokenizer
	k1        float64 // 词频饱和度参数，默认 1.5
	b         float64 // 长度归一化参数，默认 0.75
}

// NewBM25Searcher 创建 BM25 搜索器
func NewBM25Searcher(index *InvertedIndex, tokenizer *Tokenizer) *BM25Searcher {
	return &BM25Searcher{
		index:     index,
		tokenizer: tokenizer,
		k1:        1.5, // 经典值 1.2-2.0
		b:         0.75, // 经典值 0.75
	}
}

// NewBM25SearcherWithParams 创建带自定义参数的 BM25 搜索器
func NewBM25SearcherWithParams(index *InvertedIndex, tokenizer *Tokenizer, k1, b float64) *BM25Searcher {
	return &BM25Searcher{
		index:     index,
		tokenizer: tokenizer,
		k1:        k1,
		b:         b,
	}
}

// Search 执行 BM25 搜索
func (s *BM25Searcher) Search(query string, topK int) []*SearchResult {
	// 分词查询
	queryTerms := s.tokenizer.Tokenize(query)
	if len(queryTerms) == 0 {
		return make([]*SearchResult, 0)
	}

	// 获取包含查询词的所有文档
	candidateDocs := make(map[int64]bool)
	for _, term := range queryTerms {
		postingList := s.index.GetPostingList(term)
		for _, posting := range postingList {
			candidateDocs[posting.DocID] = true
		}
	}

	// 计算每个候选文档的 BM25 分数
	docScores := make(map[int64]float64)
	for docID := range candidateDocs {
		score := s.computeDocumentScore(queryTerms, docID)
		docScores[docID] = score
	}

	// 创建结果列表并排序
	results := make([]*SearchResult, 0, len(docScores))
	for docID, score := range docScores {
		results = append(results, &SearchResult{
			DocID: docID,
			Score: score,
		})
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 返回前K个结果
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// computeBM25 计算单个词的 BM25 分数
func (s *BM25Searcher) computeBM25(term string, docID int64) float64 {
	// 获取词频 (TF)
	tf := s.index.GetTermFrequency(term, docID)
	if tf == 0 {
		return 0
	}

	// 获取文档长度和平均文档长度
	docLength := s.index.GetDocumentLength(docID)
	avgDocLength := s.index.GetAverageDocumentLength()

	if avgDocLength == 0 {
		return 0
	}

	// 计算 IDF
	idf := s.computeIDF(term)
	if idf == 0 {
		return 0
	}

	// BM25 公式
	// BM25(qi,d) = IDF(qi) × (f(qi,d) × (k1 + 1)) / (f(qi,d) + k1 × (1 - b + b × |d|/avgdl))
	numerator := float64(tf) * (s.k1 + 1)
	denominator := float64(tf) + s.k1*(1-s.b+s.b*float64(docLength)/avgDocLength)

	bm25Score := idf * (numerator / denominator)
	return bm25Score
}

// computeIDF 计算 IDF（BM25 版本）
func (s *BM25Searcher) computeIDF(term string) float64 {
	df := s.index.GetDocumentFrequency(term)
	totalDocs := s.index.GetTotalDocs()

	if df == 0 || totalDocs == 0 {
		return 0
	}

	// BM25 IDF 公式：log((N - df + 0.5) / (df + 0.5))
	// 但对于小数据集，我们使用更平滑的版本：log(N/df)
	var idf float64
	if df == totalDocs {
		// 如果词在所有文档中都出现，给予最小的IDF
		idf = 0.1
	} else {
		idf = math.Log(float64(totalDocs) / float64(df))
	}

	return idf
}

// computeDocumentScore 计算查询与文档的总 BM25 分数
func (s *BM25Searcher) computeDocumentScore(queryTerms []string, docID int64) float64 {
	var totalScore float64

	// 获取文档长度信息
	docLength := s.index.GetDocumentLength(docID)
	if docLength == 0 {
		return 0
	}

	
	// 获取平均文档长度
	avgDocLength := s.index.GetAverageDocumentLength()
	if avgDocLength == 0 {
		return 0
	}

	for _, term := range queryTerms {
		// 获取词频
		tf := s.index.GetTermFrequency(term, docID)
		if tf == 0 {
			continue
		}

		// 计算 IDF
		idf := s.computeIDF(term)
		if idf == 0 {
			continue
		}

		// 基础 BM25 分数计算
		numerator := float64(tf) * (s.k1 + 1)
		denominator := float64(tf) + s.k1*(1-s.b+s.b*float64(docLength)/avgDocLength)
		bm25Score := idf * (numerator / denominator)

		// 应用字段权重（标题中的词权重更高）
		postingList := s.index.GetPostingList(term)
		var titleFreq, abstractFreq int

		for _, posting := range postingList {
			if posting.DocID == docID {
				titleFreq = posting.TitleFreq
				abstractFreq = posting.AbstractFreq
				break
			}
		}

		// 计算字段权重因子
		titleWeightFactor := 2.0 // 标题权重更高
		abstractWeightFactor := 1.0

		// 如果词在标题中，应用更高权重
		if titleFreq > 0 {
			titleProportion := float64(titleFreq) / float64(tf)
			bm25Score *= (1 + (titleWeightFactor-1)*titleProportion)
		}

		// 如果词在摘要中，应用正常权重
		if abstractFreq > 0 {
			abstractProportion := float64(abstractFreq) / float64(tf)
			bm25Score *= (1 + (abstractWeightFactor-1)*abstractProportion)
		}

		totalScore += bm25Score
	}

	return totalScore
}

// SetParameters 设置 BM25 参数
func (s *BM25Searcher) SetParameters(k1, b float64) {
	s.k1 = k1
	s.b = b
}

// GetParameters 获取当前 BM25 参数
func (s *BM25Searcher) GetParameters() (k1, b float64) {
	return s.k1, s.b
}

// SearchWithPapers 执行搜索并返回包含论文信息的完整结果
func (s *BM25Searcher) SearchWithPapers(query string, topK int, papers []*models.Paper) []*SearchResult {
	results := s.Search(query, topK)

	// 为结果添加论文信息
	paperMap := make(map[int64]*models.Paper)
	for i, paper := range papers {
		paperMap[int64(i+1)] = paper // 使用与索引相同的ID映射
	}

	for _, result := range results {
		if paper, exists := paperMap[result.DocID]; exists {
			result.Paper = paper
		}
	}

	return results
}