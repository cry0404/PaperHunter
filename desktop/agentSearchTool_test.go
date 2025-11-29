package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestAgentSearchTool(t *testing.T) {
	ast := NewAgentSearchTool()
	if ast == nil {
		t.Fatal("Failed to create AgentSearchTool")
	}

	ctx := context.Background()

	// 测试获取搜索上下文
	t.Run("GetSearchContext", func(t *testing.T) {
		context, err := ast.GetSearchContext(ctx)
		if err != nil {
			t.Fatalf("Failed to get search context: %v", err)
		}

		if context == nil {
			t.Fatal("Search context is nil")
		}

		// 验证基本数据
		if len(context.AvailableVenues) == 0 {
			t.Error("No venues available")
		}

		if len(context.ArxivCategories) == 0 {
			t.Error("No arXiv categories available")
		}

		if len(context.TrendingKeywords) == 0 {
			t.Error("No trending keywords available")
		}

		fmt.Printf("✓ Found %d venues, %d categories, %d trending keywords\n",
			len(context.AvailableVenues), len(context.ArxivCategories), len(context.TrendingKeywords))
	})

	// 测试查询分析
	testQueries := []string{
		"agent",
		"transformer",
		"neural networks",
		"ICLR conference",
		"arxiv:cs.AI",
		"reinforcement learning",
	}

	for _, query := range testQueries {
		t.Run(fmt.Sprintf("AnalyzeQuery_%s", query), func(t *testing.T) {
			enhancedQuery, err := ast.AnalyzeQuery(ctx, query)
			if err != nil {
				t.Fatalf("Failed to analyze query '%s': %v", query, err)
			}

			if enhancedQuery == nil {
				t.Fatal("Enhanced query is nil")
			}

			// 验证基本字段
			if enhancedQuery.OriginalQuery != query {
				t.Errorf("Original query mismatch: got %s, want %s", enhancedQuery.OriginalQuery, query)
			}

			if enhancedQuery.SearchStrategy == "" {
				t.Error("Search strategy is empty")
			}

			if enhancedQuery.Confidence < 0 || enhancedQuery.Confidence > 1 {
				t.Errorf("Invalid confidence score: %f", enhancedQuery.Confidence)
			}

			fmt.Printf("✓ Query '%s': Strategy=%s, Confidence=%.2f, Venue=%s, Categories=%v\n",
				query, enhancedQuery.SearchStrategy, enhancedQuery.Confidence,
				enhancedQuery.OpenReviewVenue, enhancedQuery.RecommendedCategories)

			// 如果有 arXiv 查询，打印出来
			if enhancedQuery.ArxivQuery != "" {
				fmt.Printf("  arXiv Query: %s\n", enhancedQuery.ArxivQuery)
			}

			// 如果有扩展关键词，打印出来
			if len(enhancedQuery.ExpandedKeywords) > 0 {
				fmt.Printf("  Expanded Keywords: %v\n", enhancedQuery.ExpandedKeywords)
			}
		})
	}

	// 测试搜索建议
	t.Run("GetSearchSuggestions", func(t *testing.T) {
		suggestions, err := ast.GetSearchSuggestion(ctx, "agent")
		if err != nil {
			t.Fatalf("Failed to get search suggestions: %v", err)
		}

		if len(suggestions) == 0 {
			t.Error("No suggestions provided")
		}

		fmt.Printf("✓ Got %d suggestions for 'agent'\n", len(suggestions))
		for i, suggestion := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, suggestion)
		}
	})

	// 测试导出搜索上下文
	t.Run("ExportSearchContext", func(t *testing.T) {
		contextJSON, err := ast.ExportSearchContext(ctx)
		if err != nil {
			t.Fatalf("Failed to export search context: %v", err)
		}

		if contextJSON == "" {
			t.Error("Exported context is empty")
		}

		// 验证是否是有效的 JSON
		var contextData map[string]interface{}
		if err := json.Unmarshal([]byte(contextJSON), &contextData); err != nil {
			t.Errorf("Exported context is not valid JSON: %v", err)
		}

		fmt.Printf("✓ Exported search context (%d bytes)\n", len(contextJSON))
	})
}

func TestVenueMatching(t *testing.T) {
	ast := NewAgentSearchTool()
	ctx := context.Background()

	// 获取搜索上下文
	searchContext, err := ast.GetSearchContext(ctx)
	if err != nil {
		t.Fatalf("Failed to get search context: %v", err)
	}

	testCases := []struct {
		query    string
		expected string // 预期的会议ID（部分匹配）
	}{
		{"neurips", "NeurIPS"},
		{"ICLR", "ICLR"},
		{"ICML", "ICML"},
		{"ACL", "ACL"},
		{"machine learning", "ICML"}, // 应该匹配机器学习相关会议
		{"natural language", "ACL"},  // 应该匹配 NLP 相关会议
		{"computer vision", "NeurIPS"}, // 应该匹配 CV 相关会议
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("MatchVenue_%s", tc.query), func(t *testing.T) {
			matchedVenue := ast.matchVenue(tc.query, searchContext.AvailableVenues)

			if matchedVenue == "" && tc.expected != "" {
				t.Errorf("No venue matched for query '%s', expected something with '%s'", tc.query, tc.expected)
			} else if matchedVenue != "" && tc.expected != "" {
				if !contains(matchedVenue, tc.expected) {
					t.Errorf("Venue mismatch for query '%s': got %s, expected containing '%s'", tc.query, matchedVenue, tc.expected)
				}
			}

			fmt.Printf("✓ Query '%s' -> Venue: %s\n", tc.query, matchedVenue)
		})
	}
}

func TestCategoryMatching(t *testing.T) {
	ast := NewAgentSearchTool()
	ctx := context.Background()

	// 获取搜索上下文
	searchContext, err := ast.GetSearchContext(ctx)
	if err != nil {
		t.Fatalf("Failed to get search context: %v", err)
	}

	testCases := []struct {
		query    string
		expected []string // 预期的分类
	}{
		{"artificial intelligence", []string{"cs.AI"}},
		{"machine learning", []string{"cs.LG"}},
		{"computer vision", []string{"cs.CV"}},
		{"nlp", []string{"cs.CL"}},
		{"robotics", []string{"cs.RO"}},
		{"database", []string{"cs.DB"}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("MatchCategory_%s", tc.query), func(t *testing.T) {
			queryTokens := []string{tc.query}
			matchedCategories := ast.matchCategories(queryTokens, searchContext.ArxivCategories)

			if len(matchedCategories) == 0 && len(tc.expected) > 0 {
				t.Errorf("No categories matched for query '%s', expected %v", tc.query, tc.expected)
			}

			fmt.Printf("✓ Query '%s' -> Categories: %v\n", tc.query, matchedCategories)
		})
	}
}

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			 indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}