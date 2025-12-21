package ir

import (
	"PaperHunter/internal/models"
	"fmt"
	"sync"
)

// IRSearcher IR 搜索引擎主类
type IRSearcher struct {
	index        *InvertedIndex
	tokenizer    *Tokenizer
	tfidfSearcher *TFIDFSearcher
	bm25Searcher  *BM25Searcher
	papers       []*models.Paper // 存储论文数据
	mutex        sync.RWMutex    // 保护论文数据
}

// SearchOptions 搜索选项
type SearchOptions struct {
	Query         string  // 查询字符串
	TopK          int     // 返回结果数量
	Algorithm     string  // 算法类型: "tfidf", "bm25"
	TitleWeight   float64 // 标题权重，默认 2.0
	AbstractWeight float64 // 摘要权重，默认 1.0
}

// NewIRSearcher 创建 IR 搜索引擎
func NewIRSearcher(tokenizer *Tokenizer) *IRSearcher {
	// 创建索引
	index := NewInvertedIndex(tokenizer)

	// 创建搜索器
	tfidfSearcher := NewTFIDFSearcher(index, tokenizer)
	bm25Searcher := NewBM25Searcher(index, tokenizer)

	return &IRSearcher{
		index:         index,
		tokenizer:     tokenizer,
		tfidfSearcher: tfidfSearcher,
		bm25Searcher:  bm25Searcher,
		papers:        make([]*models.Paper, 0),
	}
}

// BuildIndex 从论文列表构建索引
func (s *IRSearcher) BuildIndex(papers []*models.Paper) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(papers) == 0 {
		return fmt.Errorf("论文列表为空")
	}

	// 保存论文数据
	s.papers = make([]*models.Paper, len(papers))
	copy(s.papers, papers)

	// 批量添加文档到索引
	s.index.AddDocuments(papers)

	return nil
}

// Search 执行搜索
func (s *IRSearcher) Search(opts SearchOptions) ([]*SearchResult, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 验证参数
	if opts.Query == "" {
		return nil, fmt.Errorf("查询字符串不能为空")
	}

	if len(s.papers) == 0 {
		return nil, fmt.Errorf("索引为空，请先构建索引")
	}

	// 设置默认值
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	if opts.Algorithm == "" {
		opts.Algorithm = "bm25" // 默认使用 BM25
	}

	// 根据算法类型执行搜索
	var results []*SearchResult

	switch opts.Algorithm {
	case "tfidf":
		results = s.tfidfSearcher.SearchWithPapers(opts.Query, opts.TopK, s.papers)
	case "bm25":
		results = s.bm25Searcher.SearchWithPapers(opts.Query, opts.TopK, s.papers)
	default:
		return nil, fmt.Errorf("不支持的算法类型: %s", opts.Algorithm)
	}

	return results, nil
}

// SearchMultiple 执行多种算法的搜索，便于对比
func (s *IRSearcher) SearchMultiple(query string, topK int, algorithms []string) (map[string][]*SearchResult, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if query == "" {
		return nil, fmt.Errorf("查询字符串不能为空")
	}

	if len(s.papers) == 0 {
		return nil, fmt.Errorf("索引为空，请先构建索引")
	}

	if len(algorithms) == 0 {
		algorithms = []string{"tfidf", "bm25"}
	}

	results := make(map[string][]*SearchResult)

	for _, algorithm := range algorithms {
		switch algorithm {
		case "tfidf":
			results[algorithm] = s.tfidfSearcher.SearchWithPapers(query, topK, s.papers)
		case "bm25":
			results[algorithm] = s.bm25Searcher.SearchWithPapers(query, topK, s.papers)
		default:
			return nil, fmt.Errorf("不支持的算法类型: %s", algorithm)
		}
	}

	return results, nil
}

// AddDocument 添加单个文档到索引
func (s *IRSearcher) AddDocument(paper *models.Paper) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if paper == nil {
		return fmt.Errorf("论文不能为空")
	}

	// 生成新的文档ID
	docID := int64(len(s.papers) + 1)

	// 添加到论文列表
	s.papers = append(s.papers, paper)

	// 添加到索引
	s.index.AddDocument(docID, paper)

	return nil
}

// GetIndexStats 获取索引统计信息
func (s *IRSearcher) GetIndexStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := make(map[string]interface{})

	stats["total_papers"] = len(s.papers)
	stats["vocabulary_size"] = s.index.GetVocabularySize()
	stats["total_docs"] = s.index.GetTotalDocs()
	stats["average_doc_length"] = s.index.GetAverageDocumentLength()

	// 获取 BM25 参数
	k1, b := s.bm25Searcher.GetParameters()
	stats["bm25_k1"] = k1
	stats["bm25_b"] = b

	return stats
}

// SetBM25Parameters 设置 BM25 参数
func (s *IRSearcher) SetBM25Parameters(k1, b float64) {
	s.bm25Searcher.SetParameters(k1, b)
}

// GetBM25Parameters 获取 BM25 参数
func (s *IRSearcher) GetBM25Parameters() (k1, b float64) {
	return s.bm25Searcher.GetParameters()
}

// GetPapers 获取所有论文数据
func (s *IRSearcher) GetPapers() []*models.Paper {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	papers := make([]*models.Paper, len(s.papers))
	copy(papers, s.papers)
	return papers
}

// GetPaperByID 根据 ID 获取论文
func (s *IRSearcher) GetPaperByID(docID int64) *models.Paper {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if docID <= 0 || int(docID) > len(s.papers) {
		return nil
	}

	return s.papers[docID-1] // ID 从 1 开始，数组索引从 0 开始
}

// IsIndexEmpty 检查索引是否为空
func (s *IRSearcher) IsIndexEmpty() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.papers) == 0
}

// ClearIndex 清空索引
func (s *IRSearcher) ClearIndex() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 创建新的索引
	s.index = NewInvertedIndex(s.tokenizer)
	s.tfidfSearcher = NewTFIDFSearcher(s.index, s.tokenizer)
	s.bm25Searcher = NewBM25Searcher(s.index, s.tokenizer)

	// 清空论文数据
	s.papers = make([]*models.Paper, 0)
}

// IsEmpty 检查索引是否为空
func (s *IRSearcher) IsEmpty() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.papers) == 0
}