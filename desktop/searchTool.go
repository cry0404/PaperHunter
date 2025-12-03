package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"PaperHunter/internal/core"
	"PaperHunter/internal/models"
	"PaperHunter/pkg/logger"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// SearchExample 示例论文（用于语义搜索）
type SearchExample struct {
	Title    string `json:"title" jsonschema:"description=Title of the example paper"`
	Abstract string `json:"abstract" jsonschema:"description=Abstract of the example paper"`
}

// SearchInput 搜索工具的输入参数
type SearchInput struct {
	// Query 查询文本（用于语义搜索或关键词搜索）
	// 注意：query 和 examples 至少需要提供一个
	Query string `json:"query,omitempty" jsonschema:"description=Search query text for semantic or keyword search. REQUIRED if examples is not provided"`

	// Examples 示例论文列表（用于基于示例的语义搜索）

	Examples []SearchExample `json:"examples,omitempty" jsonschema:"description=Example papers for similarity-based search. REQUIRED if query is not provided"`

	// Semantic 是否使用语义搜索（默认 true）
	Semantic bool `json:"semantic,omitempty" jsonschema:"description=Whether to use semantic search (default: true)"`

	// TopK 返回前 K 个最相似的结果
	TopK int `json:"top_k,omitempty" jsonschema:"description=Number of top similar papers to return"`

	// Limit 数据库查询数量限制
	Limit int `json:"limit,omitempty" jsonschema:"description=Database query limit (0 means no limit)"`

	// Source 数据源过滤（如 arxiv, openreview, acl 等）
	Source string `json:"source,omitempty" jsonschema:"description=Filter by data source (e.g., arxiv, openreview, acl)"`

	// DateFrom 开始日期，格式 YYYY-MM-DD
	DateFrom string `json:"date_from,omitempty" jsonschema:"description=Start date in YYYY-MM-DD format"`

	// DateTo 结束日期，格式 YYYY-MM-DD
	DateTo string `json:"date_to,omitempty" jsonschema:"description=End date in YYYY-MM-DD format"`

	// ComputeEmbed 搜索前是否先计算缺失的 embedding
	ComputeEmbed bool `json:"compute_embed,omitempty" jsonschema:"description=Compute missing embeddings before search"`

	// EmbedBatch embedding 批量计算数量
	EmbedBatch int `json:"embed_batch,omitempty" jsonschema:"description=Batch size for computing embeddings"`
}

// SearchOutput 搜索工具的输出结果
type SearchOutput struct {
	Count   int                    `json:"count" jsonschema:"description=Number of papers found"`
	Papers  []*models.SimilarPaper `json:"papers" jsonschema:"description=List of similar papers with similarity scores"`
	Message string                 `json:"message" jsonschema:"description=Result message"`
}

// NewSearchTool 创建搜索工具，接受 App 实例
func NewSearchTool(app *App) tool.InvokableTool {
	searchTool, err := utils.InferTool("search", `Search papers in the local database using semantic search or keyword search.

**REQUIRED PARAMETERS (must provide at least one):**
- query: Search query text (equivalent to CLI --query="...") - REQUIRED if examples is not provided
- examples: List of example papers for similarity-based search - REQUIRED if query is not provided
  Each example should have: {title: string, abstract: string}

**OPTIONAL PARAMETERS:**
- top_k: Number of top similar papers to return (equivalent to CLI --top-k=N, default: 100)
- limit: Database query limit (equivalent to CLI --limit=N, 0 means no limit)
- source: Filter by data source (equivalent to CLI --source=arxiv)
- date_from: Start date in YYYY-MM-DD format (equivalent to CLI --from=YYYY-MM-DD)
- date_to: End date in YYYY-MM-DD format (equivalent to CLI --until=YYYY-MM-DD)
- semantic: Whether to use semantic search (default: true)

**IMPORTANT:** 
- You MUST provide either 'query' OR 'examples' parameter. The tool will fail if both are missing.
- **DO NOT use this tool for Zotero-based recommendations. Use zotero_recommend tool instead.**
- Use top_k (not limit) to control the number of results returned.
- If top_k is not specified but limit is provided, limit will be used as top_k.
- Date format must be YYYY-MM-DD (e.g., "2025-11-03", not "2025-11-03T00:00:00Z").
- When searching for papers similar to Zotero papers, use 'examples' parameter with the Zotero paper's title and abstract.`, func(ctx context.Context, input *SearchInput) (output *SearchOutput, err error) {
		if app == nil || app.coreApp == nil {
			return nil, fmt.Errorf("app instance is not initialized")
		}

		// 如果设置了计算 embedding，先计算缺失的向量
		if input.ComputeEmbed {
			batch := input.EmbedBatch
			if batch <= 0 {
				batch = 100
			}
			_, err := app.coreApp.ComputeMissingEmbeddings(ctx, batch)
			if err != nil {
				return nil, fmt.Errorf("failed to compute embeddings: %w", err)
			}
		}

		// 构建 SearchCondition
		cond := models.SearchCondition{
			Limit: input.Limit,
		}

		if input.Source != "" {
			cond.Sources = []string{input.Source}
		}

		if input.DateFrom != "" {
			t, err := time.Parse("2006-01-02", input.DateFrom)
			if err != nil {
				return nil, fmt.Errorf("invalid date_from format: %w", err)
			}
			cond.DateFrom = &t
		}

		if input.DateTo != "" {
			t, err := time.Parse("2006-01-02", input.DateTo)
			if err != nil {
				return nil, fmt.Errorf("invalid date_to format: %w", err)
			}
			cond.DateTo = &t
		}

		var examples []*models.Paper
		if len(input.Examples) > 0 {
			for _, e := range input.Examples {
				p := &models.Paper{
					Title:    e.Title,
					Abstract: e.Abstract,
				}
				if p.Title != "" || p.Abstract != "" {
					examples = append(examples, p)
				}
			}
		}

		// 设置 TopK 默认值（与命令行一致）
		topK := input.TopK
		if topK <= 0 {
			// 如果 TopK 未设置，使用 Limit 作为备选，否则使用默认值 100
			if input.Limit > 0 {
				topK = input.Limit
			} else {
				topK = 100
			}
		}

		if strings.TrimSpace(input.Query) == "" && len(examples) == 0 {
			logger.Warn("search 工具调用失败：缺少必需参数。query='%s', examples_count=%d", input.Query, len(examples))
			return &SearchOutput{
				Count:   0,
				Papers:  nil,
				Message: "错误：请提供查询文本(query)或示例论文(examples)参数。如果基于 Zotero 推荐，请使用 zotero_recommend 工具（action: daily_recommend）",
			}, fmt.Errorf("请提供查询文本(query)或示例论文(examples)。如果基于 Zotero 推荐，请使用 zotero_recommend 工具")
		}


		opts := core.SearchOptions{
			Query:     input.Query,
			Examples:  examples,
			Condition: cond,
			TopK:      topK,
			Semantic:  input.Semantic,
		}


		logger.Info("搜索参数: query=%s, examples_count=%d, source=%s, date_from=%s, date_to=%s, top_k=%d, limit=%d, semantic=%v",
			input.Query, len(examples), input.Source, input.DateFrom, input.DateTo, topK, input.Limit, input.Semantic)


		results, err := app.coreApp.Search(ctx, opts)
		if err != nil {
			return &SearchOutput{
				Count:   0,
				Papers:  nil,
				Message: fmt.Sprintf("Search failed: %v", err),
			}, err
		}

		return &SearchOutput{
			Count:   len(results),
			Papers:  results,
			Message: fmt.Sprintf("Successfully found %d papers", len(results)),
		}, nil
	})

	if err != nil {
		log.Fatalf("failed to create search tool: %v", err)
	}

	return searchTool
}
