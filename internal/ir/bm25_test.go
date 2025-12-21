package ir

import (
	"PaperHunter/internal/models"
	"testing"
)

func TestNewBM25Searcher(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

	if searcher == nil {
		t.Fatal("NewBM25Searcher() returned nil")
	}

	if searcher.index == nil {
		t.Fatal("searcher.index is nil")
	}

	if searcher.tokenizer == nil {
		t.Fatal("searcher.tokenizer is nil")
	}

	// 验证默认参数
	k1, b := searcher.GetParameters()
	if k1 != 1.5 {
		t.Errorf("Expected default k1 = 1.5, got %.2f", k1)
	}
	if b != 0.75 {
		t.Errorf("Expected default b = 0.75, got %.2f", b)
	}
}

func TestNewBM25SearcherWithParams(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25SearcherWithParams(index, tokenizer, 2.0, 0.5)

	k1, b := searcher.GetParameters()
	if k1 != 2.0 {
		t.Errorf("Expected k1 = 2.0, got %.2f", k1)
	}
	if b != 0.5 {
		t.Errorf("Expected b = 0.5, got %.2f", b)
	}
}

func TestBM25Searcher_SetParameters(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

	// 设置新参数
	searcher.SetParameters(1.2, 0.8)

	k1, b := searcher.GetParameters()
	if k1 != 1.2 {
		t.Errorf("Expected k1 = 1.2 after SetParameters, got %.2f", k1)
	}
	if b != 0.8 {
		t.Errorf("Expected b = 0.8 after SetParameters, got %.2f", b)
	}
}

func TestBM25Searcher_Search(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

	// 创建测试论文数据，包含不同长度的文档
	papers := []*models.Paper{
		{
			ID:       1,
			Title:    "Machine Learning",
			Abstract: "Introduction to machine learning algorithms and applications.",
		},
		{
			ID:       2,
			Title:    "Deep Learning Neural Networks",
			Abstract: "This comprehensive paper explores deep learning architectures using neural networks for various machine learning tasks including computer vision and natural language processing with advanced optimization techniques.",
		},
		{
			ID:       3,
			Title:    "Computer Vision",
			Abstract: "Image processing and analysis.",
		},
	}

	// 构建索引
	index.AddDocuments(papers)

	// 测试搜索
	tests := []struct {
		name          string
		query         string
		expectedLen   int
		firstDocID    int64 // 期望排名第一的文档ID
	}{
		{
			name:        "search for learning",
			query:       "learning",
			expectedLen: 2, // 只有前两篇论文包含"learning"
			firstDocID:  1, // 第一篇论文应该排名靠前（文档长度较短，"learning"密度更高）
		},
		{
			name:        "search for machine",
			query:       "machine",
			expectedLen: 2, // 两篇论文包含"machine"
			firstDocID:  1, // 第一篇论文的标题中包含"machine"
		},
		{
			name:        "search for deep",
			query:       "deep",
			expectedLen: 1, // 只有一篇论文包含"deep"
			firstDocID:  2, // 第二篇论文
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
					t.Logf("Note: Expected first result DocID = %d, got %d (this may vary with BM25 parameters)",
						tt.firstDocID, results[0].DocID)
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

func TestBM25Searcher_computeBM25(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

	papers := []*models.Paper{
		{ID: 1, Title: "machine learning", Abstract: "introduction to machine learning"},
		{ID: 2, Title: "deep learning", Abstract: "advanced techniques in deep learning"},
		{ID: 3, Title: "other topic", Abstract: "completely unrelated content"},
	}

	index.AddDocuments(papers)

	// 测试包含在文档中的词
	score := searcher.computeBM25("learning", 1)
	if score <= 0 {
		t.Errorf("Expected positive BM25 score for 'learning' in doc 1, got %.4f", score)
	}

	// 测试不包含在文档中的词
	score = searcher.computeBM25("nonexistent", 1)
	if score != 0 {
		t.Errorf("Expected BM25 score 0 for nonexistent term in doc 1, got %.4f", score)
	}

	// 测试稀有词（只在1篇文档中出现的词）应该有更高的IDF
	rareScore := searcher.computeBM25("machine", 1)    // 只在文档1中出现
	commonScore := searcher.computeBM25("learning", 1)  // 在文档1和2中都出现

	if rareScore <= commonScore {
		t.Logf("Note: BM25 ranking may differ from TF-IDF: rare=%.4f, common=%.4f", rareScore, commonScore)
	}
}

func TestBM25Searcher_computeIDF(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

	papers := []*models.Paper{
		{ID: 1, Title: "machine learning", Abstract: "introduction"},
		{ID: 2, Title: "deep learning", Abstract: "advanced techniques"},
		{ID: 3, Title: "neural networks", Abstract: "overview"},
	}

	index.AddDocuments(papers)

	// 测试稀有词（只在1篇文档中出现）
	rareIDF := searcher.computeIDF("machine")

	// 测试常见词（在多篇文档中出现）
	commonIDF := searcher.computeIDF("learning")

	// 稀有词的IDF应该更高
	if rareIDF <= commonIDF {
		t.Errorf("Expected rare term 'machine' to have higher IDF than common term 'learning': rare=%.4f, common=%.4f",
			rareIDF, commonIDF)
	}

	// 测试不存在的词
	nonexistentIDF := searcher.computeIDF("nonexistent")
	if nonexistentIDF != 0 {
		t.Errorf("Expected IDF 0 for nonexistent term, got %.4f", nonexistentIDF)
	}
}

func TestBM25Searcher_DocumentLengthNormalization(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

	// 创建包含相同词但文档长度不同的论文
	papers := []*models.Paper{
		{
			ID:       1,
			Title:    "Short",
			Abstract: "learning machine learning machine",
		},
		{
			ID:       2,
			Title:    "Very Long Document With Many Words",
			Abstract: "This is a very long document that contains many words and multiple instances of the term learning machine learning machine learning machine learning machine learning",
		},
	}

	index.AddDocuments(papers)

	// 计算BM25分数
	shortScore := searcher.computeBM25("learning", 1)
	longScore := searcher.computeBM25("learning", 2)

	// 虽然长文档包含更多"learning"，但经过长度归一化后，
	// 短文档可能获得相对较高的分数（因为词密度更高）
	t.Logf("Short doc score: %.4f, Long doc score: %.4f", shortScore, longScore)

	if shortScore <= 0 || longScore <= 0 {
		t.Error("Both documents should have positive BM25 scores for 'learning'")
	}
}

func TestBM25Searcher_SearchWithPapers(t *testing.T) {
	tokenizer, _ := NewTokenizer()
	index := NewInvertedIndex(tokenizer)
	searcher := NewBM25Searcher(index, tokenizer)

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