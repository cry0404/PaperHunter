package ir

import (
	"PaperHunter/internal/models"
	"testing"
)

func TestNewTFIDFSearcher(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewTFIDFSearcher(index, tokenizer)

	if searcher == nil {
		t.Fatal("NewTFIDFSearcher() returned nil")
	}

	if searcher.index == nil {
		t.Fatal("searcher.index is nil")
	}

	if searcher.tokenizer == nil {
		t.Fatal("searcher.tokenizer is nil")
	}
}

func TestTFIDFSearcher_Search(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewTFIDFSearcher(index, tokenizer)

	// 创建测试论文数据
	papers := []*models.Paper{
		{
			ID:       1,
			Title:    "Deep Learning Neural Networks",
			Abstract: "This paper discusses deep learning and neural networks for machine learning tasks.",
		},
		{
			ID:       2,
			Title:    "Machine Learning Algorithms",
			Abstract: "Introduction to various machine learning algorithms and their applications.",
		},
		{
			ID:       3,
			Title:    "Computer Vision Systems",
			Abstract: "Computer vision techniques using image processing and pattern recognition.",
		},
	}

	// 构建索引
	index.AddDocuments(papers)

	// 测试搜索
	tests := []struct {
		name          string
		query         string
		expectedLen   int
		firstDocID    int64 // 期望排名第一的文档ID（包含查询词最多的）
	}{
		{
			name:        "search for learning",
			query:       "learning",
			expectedLen: 2, // 两篇论文包含"learning"
			firstDocID:  1, // 第一篇论文的标题中包含"learning"
		},
		{
			name:        "search for deep",
			query:       "deep",
			expectedLen: 1, // 只有一篇论文包含"deep"
			firstDocID:  1, // 第一篇论文
		},
		{
			name:        "search for vision",
			query:       "vision",
			expectedLen: 1, // 只有一篇论文包含"vision"
			firstDocID:  3, // 第三篇论文
		},
		{
			name:        "search for nonexistent term",
			query:       "nonexistent",
			expectedLen: 0, // 没有论文包含这个词
		},
		{
			name:        "empty query",
			query:       "",
			expectedLen: 0, // 空查询应该返回空结果
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searcher.Search(tt.query, 10)

			if len(results) != tt.expectedLen {
				t.Errorf("Expected %d results, got %d", tt.expectedLen, len(results))
			}

			// 验证结果排序（分数应该降序）
			for i := 1; i < len(results); i++ {
				if results[i-1].Score < results[i].Score {
					t.Errorf("Results not properly sorted: score[%d] = %.4f < score[%d] = %.4f",
						i-1, results[i-1].Score, i, results[i].Score)
				}
			}

			// 验证第一个结果的DocID（如果有结果）
			if len(results) > 0 && tt.firstDocID > 0 {
				if results[0].DocID != tt.firstDocID {
					t.Errorf("Expected first result DocID = %d, got %d", tt.firstDocID, results[0].DocID)
				}
			}

			// 验证分数合理性
			for _, result := range results {
				if result.Score <= 0 {
					t.Errorf("Expected positive score, got %.4f for doc %d", result.Score, result.DocID)
				}
			}
		})
	}
}

func TestTFIDFSearcher_SearchWithTopK(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewTFIDFSearcher(index, tokenizer)

	// 创建测试论文数据
	papers := []*models.Paper{
		{ID: 1, Title: "Machine Learning", Abstract: "Introduction to machine learning."},
		{ID: 2, Title: "Deep Learning", Abstract: "Advanced deep learning techniques."},
		{ID: 3, Title: "Learning Algorithms", Abstract: "Various learning algorithms."},
		{ID: 4, Title: "Neural Networks", Abstract: "Neural network architectures."},
	}

	index.AddDocuments(papers)

	// 测试TopK限制
	results := searcher.Search("learning", 2)

	if len(results) > 2 {
		t.Errorf("Expected at most 2 results, got %d", len(results))
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 results for 'learning', got %d", len(results))
	}

	// 验证结果包含正确的文档（前3篇论文都包含"learning"）
	docIDs := make(map[int64]bool)
	for _, result := range results {
		docIDs[result.DocID] = true
	}

	// 应该包含ID为1, 2, 3中的至少两个
	learningDocs := []int64{1, 2, 3}
	foundCount := 0
	for _, id := range learningDocs {
		if docIDs[id] {
			foundCount++
		}
	}

	if foundCount != len(results) {
		t.Errorf("Results should only contain docs with 'learning', found %d out of %d", foundCount, len(results))
	}
}

func TestTFIDFSearcher_SearchWithPapers(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewTFIDFSearcher(index, tokenizer)

	papers := []*models.Paper{
		{ID: 1, Title: "Machine Learning", Abstract: "Introduction to machine learning."},
		{ID: 2, Title: "Deep Learning", Abstract: "Advanced deep learning techniques."},
	}

	index.AddDocuments(papers)

	results := searcher.SearchWithPapers("learning", 10, papers)

	if len(results) == 0 {
		t.Error("Expected search results, got empty slice")
	}

	// 验证结果包含论文信息
	for _, result := range results {
		if result.Paper == nil {
			t.Errorf("Result %d missing paper information", result.DocID)
		}

		if result.Paper.Title == "" {
			t.Errorf("Result %d has empty title", result.DocID)
		}
	}
}

func TestTFIDFSearcher_computeTFIDF(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewTFIDFSearcher(index, tokenizer)

	papers := []*models.Paper{
		{ID: 1, Title: "machine learning", Abstract: "introduction to machine learning"},
		{ID: 2, Title: "deep learning", Abstract: "advanced techniques in deep learning"},
		{ID: 3, Title: "other topic", Abstract: "completely unrelated content"},
	}

	index.AddDocuments(papers)

	// 测试包含在文档中的词
	score := searcher.computeTFIDF("learning", 1)
	if score <= 0 {
		t.Errorf("Expected positive TF-IDF score for 'learning' in doc 1, got %.4f", score)
	}

	// 测试不包含在文档中的词
	score = searcher.computeTFIDF("nonexistent", 1)
	if score != 0 {
		t.Errorf("Expected TF-IDF score 0 for nonexistent term in doc 1, got %.4f", score)
	}

	// 测试稀有词（只在1篇文档中出现的词）应该有更高的IDF
	rareScore := searcher.computeTFIDF("machine", 1)    // 只在文档1中出现
	commonScore := searcher.computeTFIDF("learning", 1)  // 在文档1和2中都出现

	if rareScore <= commonScore {
		t.Errorf("Expected rare term 'machine' to have higher TF-IDF than common term 'learning': machine=%.4f, learning=%.4f",
			rareScore, commonScore)
	}
}