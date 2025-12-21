package ir

import (
	"PaperHunter/internal/models"
	"testing"
	"time"
)

func TestNewInvertedIndex(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)

	if index == nil {
		t.Fatal("NewInvertedIndex() returned nil")
	}

	if index.index == nil {
		t.Fatal("index map not initialized")
	}

	if index.docLengths == nil {
		t.Fatal("docLengths map not initialized")
	}

	if index.GetTotalDocs() != 0 {
		t.Errorf("Expected 0 docs, got %d", index.GetTotalDocs())
	}
}

func TestInvertedIndex_AddDocument(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)

	// 创建测试论文
	paper := &models.Paper{
		ID:       1,
		Title:    "Deep Learning for Computer Vision",
		Abstract: "This paper explores deep learning techniques for computer vision tasks using convolutional neural networks.",
	}

	// 添加文档
	index.AddDocument(1, paper)

	// 验证统计数据
	if index.GetTotalDocs() != 1 {
		t.Errorf("Expected 1 doc, got %d", index.GetTotalDocs())
	}

	if index.GetDocumentLength(1) == 0 {
		t.Error("Document length should not be 0")
	}

	// 验证索引内容
	if df := index.GetDocumentFrequency("deep"); df != 1 {
		t.Errorf("Expected DF(deep) = 1, got %d", df)
	}

	if tf := index.GetTermFrequency("deep", 1); tf == 0 {
		t.Error("Term frequency for 'deep' should not be 0")
	}
}

func TestInvertedIndex_AddDocuments(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)

	papers := []*models.Paper{
		{
			ID:       1,
			Title:    "Machine Learning Basics",
			Abstract: "Introduction to machine learning algorithms and concepts.",
		},
		{
			ID:       2,
			Title:    "Neural Networks",
			Abstract: "Deep learning architectures for pattern recognition.",
		},
		{
			ID:       3,
			Title:    "Computer Vision",
			Abstract: "Image processing and analysis using machine learning.",
		},
	}

	// 批量添加文档
	index.AddDocuments(papers)

	// 验证统计数据
	if index.GetTotalDocs() != 3 {
		t.Errorf("Expected 3 docs, got %d", index.GetTotalDocs())
	}

	// 验证词汇表大小
	vocabSize := index.GetVocabularySize()
	if vocabSize <= 0 {
		t.Errorf("Expected positive vocabulary size, got %d", vocabSize)
	}

	// 验证特定词的文档频率
	if df := index.GetDocumentFrequency("machine"); df != 2 {
		t.Errorf("Expected DF(machine) = 2, got %d", df)
	}

	if df := index.GetDocumentFrequency("learning"); df != 3 {
		t.Errorf("Expected DF(learning) = 3, got %d", df)
	}

	if df := index.GetDocumentFrequency("vision"); df != 1 {
		t.Errorf("Expected DF(vision) = 1, got %d", df)
	}
}

func TestInvertedIndex_GetPostingList(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)

	papers := []*models.Paper{
		{
			ID:       1,
			Title:    "Deep Learning",
			Abstract: "Neural networks and deep learning.",
		},
		{
			ID:       2,
			Title:    "Machine Learning",
			Abstract: "Statistical learning theory and applications.",
		},
	}

	index.AddDocuments(papers)

	// 测试存在的词
	postingList := index.GetPostingList("learning")
	if len(postingList) != 2 {
		t.Errorf("Expected 2 postings for 'learning', got %d", len(postingList))
	}

	// 验证posting结构
	for _, posting := range postingList {
		if posting.DocID != 1 && posting.DocID != 2 {
			t.Errorf("Unexpected DocID %d in posting list", posting.DocID)
		}

		if posting.TermFreq <= 0 {
			t.Errorf("Invalid term frequency %d for doc %d", posting.TermFreq, posting.DocID)
		}
	}

	// 测试不存在的词
	postingList = index.GetPostingList("nonexistent")
	if len(postingList) != 0 {
		t.Errorf("Expected 0 postings for 'nonexistent', got %d", len(postingList))
	}
}

func TestInvertedIndex_AverageDocumentLength(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)

	// 空索引
	if avg := index.GetAverageDocumentLength(); avg != 0 {
		t.Errorf("Expected average length 0 for empty index, got %.2f", avg)
	}

	papers := []*models.Paper{
		{
			ID:       1,
			Title:    "Short Title",
			Abstract: "Short abstract.",
		},
		{
			ID:       2,
			Title:    "A Much Longer Title With More Words",
			Abstract: "This is a considerably longer abstract with many more words to test the average calculation.",
		},
	}

	index.AddDocuments(papers)

	avg := index.GetAverageDocumentLength()
	if avg <= 0 {
		t.Errorf("Expected positive average length, got %.2f", avg)
	}

	// 验证计算是否合理
	doc1Len := index.GetDocumentLength(1)
	doc2Len := index.GetDocumentLength(2)
	expectedAvg := float64(doc1Len+doc2Len) / 2.0

	if avg != expectedAvg {
		t.Errorf("Expected average length %.2f, got %.2f", expectedAvg, avg)
	}
}

// createTestPaper 创建测试论文的辅助函数
func createTestPaper(id int64, title, abstract string) *models.Paper {
	return &models.Paper{
		ID:       id,
		Source:   "test",
		SourceID: "test-id",
		URL:      "http://example.com",
		Title:    title,
		Abstract: abstract,
		Authors:  []string{"Test Author"},
		Categories: []string{"cs.AI"},
		FirstSubmittedAt: time.Now(),
		FirstAnnouncedAt: time.Now(),
		UpdatedAt:        time.Now(),
	}
}