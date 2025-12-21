package ir

import (
	"PaperHunter/internal/models"
	"sync"
)

// Posting 倒排索引中的 posting 条目
type Posting struct {
	DocID        int64  // 文档ID
	TermFreq     int    // 总词频（标题+摘要）
	TitleFreq    int    // 标题中的词频
	AbstractFreq int    // 摘要中的词频
}

// PostingList 某个词的倒排列表
type PostingList []Posting

// InvertedIndex 倒排索引结构
type InvertedIndex struct {
	index          map[string]PostingList  // 倒排索引：term -> []Posting
	docLengths     map[int64]int           // 文档总长度
	titleLengths   map[int64]int           // 标题长度
	abstractLengths map[int64]int          // 摘要长度
	tokenizer      *Tokenizer
	mutex          sync.RWMutex           // 读写锁，保证并发安全
	totalDocs      int                    // 文档总数
	avgDocLength   float64                // 平均文档长度
}

// NewInvertedIndex 创建新的倒排索引
func NewInvertedIndex(tokenizer *Tokenizer) *InvertedIndex {
	return &InvertedIndex{
		index:           make(map[string]PostingList),
		docLengths:      make(map[int64]int),
		titleLengths:    make(map[int64]int),
		abstractLengths: make(map[int64]int),
		tokenizer:       tokenizer,
		totalDocs:       0,
		avgDocLength:    0,
	}
}

// AddDocument 添加单个文档到索引
func (ii *InvertedIndex) AddDocument(docID int64, paper *models.Paper) {
	ii.mutex.Lock()
	defer ii.mutex.Unlock()

	// 分词标题和摘要
	titleTokens := ii.tokenizer.Tokenize(paper.Title)
	abstractTokens := ii.tokenizer.Tokenize(paper.Abstract)

	// 计算标题词频
	titleTermFreqs := make(map[string]int)
	for _, token := range titleTokens {
		titleTermFreqs[token]++
	}

	// 计算摘要词频
	abstractTermFreqs := make(map[string]int)
	for _, token := range abstractTokens {
		abstractTermFreqs[token]++
	}

	// 合并所有词项
	allTerms := make(map[string]bool)
	for term := range titleTermFreqs {
		allTerms[term] = true
	}
	for term := range abstractTermFreqs {
		allTerms[term] = true
	}

	// 为每个词项创建 posting
	for term := range allTerms {
		titleFreq := titleTermFreqs[term]
		abstractFreq := abstractTermFreqs[term]
		totalFreq := titleFreq + abstractFreq

		posting := Posting{
			DocID:        docID,
			TermFreq:     totalFreq,
			TitleFreq:    titleFreq,
			AbstractFreq: abstractFreq,
		}

		// 添加到倒排索引
		if _, exists := ii.index[term]; !exists {
			ii.index[term] = make(PostingList, 0)
		}
		ii.index[term] = append(ii.index[term], posting)
	}

	// 记录文档长度
	ii.docLengths[docID] = len(titleTokens) + len(abstractTokens)
	ii.titleLengths[docID] = len(titleTokens)
	ii.abstractLengths[docID] = len(abstractTokens)

	// 更新统计信息
	ii.totalDocs++
	ii.updateAverageDocumentLength()
}

// AddDocuments 批量添加文档到索引
func (ii *InvertedIndex) AddDocuments(papers []*models.Paper) {
	for i, paper := range papers {
		docID := int64(i + 1) // 临时ID，实际应用中应该使用真实的文档ID
		ii.AddDocument(docID, paper)
	}
}

// GetPostingList 获取词的倒排列表
func (ii *InvertedIndex) GetPostingList(term string) PostingList {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	if postingList, exists := ii.index[term]; exists {
		return postingList
	}
	return make(PostingList, 0)
}

// GetDocumentFrequency 获取文档频率（DF）- 包含该词的文档数
func (ii *InvertedIndex) GetDocumentFrequency(term string) int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	if postingList, exists := ii.index[term]; exists {
		return len(postingList)
	}
	return 0
}

// GetTermFrequency 获取词频（TF）- 词在指定文档中的频率
func (ii *InvertedIndex) GetTermFrequency(term string, docID int64) int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	if postingList, exists := ii.index[term]; exists {
		for _, posting := range postingList {
			if posting.DocID == docID {
				return posting.TermFreq
			}
		}
	}
	return 0
}

// GetAverageDocumentLength 获取平均文档长度
func (ii *InvertedIndex) GetAverageDocumentLength() float64 {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	return ii.avgDocLength
}

// GetDocumentLength 获取指定文档的长度
func (ii *InvertedIndex) GetDocumentLength(docID int64) int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	if length, exists := ii.docLengths[docID]; exists {
		return length
	}
	return 0
}

// GetTitleLength 获取标题长度
func (ii *InvertedIndex) GetTitleLength(docID int64) int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	if length, exists := ii.titleLengths[docID]; exists {
		return length
	}
	return 0
}

// GetAbstractLength 获取摘要长度
func (ii *InvertedIndex) GetAbstractLength(docID int64) int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	if length, exists := ii.abstractLengths[docID]; exists {
		return length
	}
	return 0
}

// GetTotalDocs 获取文档总数
func (ii *InvertedIndex) GetTotalDocs() int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	return ii.totalDocs
}

// updateAverageDocumentLength 更新平均文档长度
func (ii *InvertedIndex) updateAverageDocumentLength() {
	if ii.totalDocs == 0 {
		ii.avgDocLength = 0
		return
	}

	totalLength := 0
	for _, length := range ii.docLengths {
		totalLength += length
	}
	ii.avgDocLength = float64(totalLength) / float64(ii.totalDocs)
}

// GetVocabularySize 获取词汇表大小
func (ii *InvertedIndex) GetVocabularySize() int {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	return len(ii.index)
}

// GetTerms 获取所有词汇
func (ii *InvertedIndex) GetTerms() []string {
	ii.mutex.RLock()
	defer ii.mutex.RUnlock()

	terms := make([]string, 0, len(ii.index))
	for term := range ii.index {
		terms = append(terms, term)
	}
	return terms
}