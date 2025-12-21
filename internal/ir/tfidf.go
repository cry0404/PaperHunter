package ir

import (
	"PaperHunter/internal/models"
	"math"
	"sort"
)

type TFIDFSearcher struct {
	index     *InvertedIndex
	tokenizer *Tokenizer
}

type SearchResult struct {
	DocID int64
	Score float64
	Paper *models.Paper
}

func NewTFIDFSearcher(index *InvertedIndex, tokenizer *Tokenizer) *TFIDFSearcher {
	return &TFIDFSearcher{
		index:     index,
		tokenizer: tokenizer,
	}
}

func (s *TFIDFSearcher) Search(query string, topK int) []*SearchResult {

	queryTerms := s.tokenizer.Tokenize(query)
	if len(queryTerms) == 0 {
		return make([]*SearchResult, 0)
	}

	candidateDocs := make(map[int64]bool)
	for _, term := range queryTerms {
		postingList := s.index.GetPostingList(term)
		for _, posting := range postingList {
			candidateDocs[posting.DocID] = true
		}
	}

	docScores := make(map[int64]float64)
	for docID := range candidateDocs {
		score := s.computeDocumentScore(queryTerms, docID)
		docScores[docID] = score
	}

	results := make([]*SearchResult, 0, len(docScores))
	for docID, score := range docScores {
		results = append(results, &SearchResult{
			DocID: docID,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

func (s *TFIDFSearcher) computeTFIDF(term string, docID int64) float64 {

	tf := s.index.GetTermFrequency(term, docID)
	if tf == 0 {
		return 0
	}

	df := s.index.GetDocumentFrequency(term)
	if df == 0 {
		return 0
	}

	totalDocs := s.index.GetTotalDocs()
	if totalDocs == 0 {
		return 0
	}

	idf := math.Log(float64(totalDocs) / float64(df))

	tfLog := 1 + math.Log(float64(tf))
	tfidf := tfLog * idf

	return tfidf
}

// computeDocumentScore 计算查询与文档的总 TF-IDF 分数
func (s *TFIDFSearcher) computeDocumentScore(queryTerms []string, docID int64) float64 {
	var totalScore float64

	for _, term := range queryTerms {

		tf := s.index.GetTermFrequency(term, docID)
		if tf == 0 {
			continue
		}

		titleFreq := 0
		abstractFreq := 0
		postingList := s.index.GetPostingList(term)

		for _, posting := range postingList {
			if posting.DocID == docID {
				titleFreq = posting.TitleFreq
				abstractFreq = posting.AbstractFreq
				break
			}
		}

		df := s.index.GetDocumentFrequency(term)
		totalDocs := s.index.GetTotalDocs()
		if totalDocs == 0 || df == 0 {
			continue
		}

		idf := math.Log(float64(totalDocs) / float64(df))

		titleWeight := 2.0
		abstractWeight := 1.0

		titleTFIDF := (1 + math.Log(float64(titleFreq))) * idf * titleWeight
		abstractTFIDF := (1 + math.Log(float64(abstractFreq))) * idf * abstractWeight

		totalScore += titleTFIDF + abstractTFIDF
	}

	return totalScore
}

// SearchWithPapers 执行搜索并返回包含论文信息的完整结果
func (s *TFIDFSearcher) SearchWithPapers(query string, topK int, papers []*models.Paper) []*SearchResult {
	results := s.Search(query, topK)

	paperMap := make(map[int64]*models.Paper)
	for i, paper := range papers {
		paperMap[int64(i+1)] = paper
	}

	for _, result := range results {
		if paper, exists := paperMap[result.DocID]; exists {
			result.Paper = paper
		}
	}

	return results
}
