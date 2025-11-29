package main

import (
	"context"
	"fmt"
	"log"

	"PaperHunter/internal/platform"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CrawlerInput 爬虫工具的输入参数
type CrawlerInput struct {
	// Platform 平台名称，如 "arxiv", "openreview", "acl" 等
	Platform string `json:"platform" jsonschema:"required,description=The platform to crawl from (e.g., arxiv, openreview, acl)"`

	// Keywords 关键词列表
	Keywords []string `json:"keywords,omitempty" jsonschema:"description=List of keywords to search for"`

	// Categories 类别列表
	Categories []string `json:"categories,omitempty" jsonschema:"description=List of categories to filter"`

	// DateFrom 开始日期，格式 YYYY-MM-DD
	DateFrom string `json:"date_from,omitempty" jsonschema:"description=Start date in YYYY-MM-DD format"`

	// DateTo 结束日期，格式 YYYY-MM-DD
	DateTo string `json:"date_to,omitempty" jsonschema:"description=End date in YYYY-MM-DD format"`

	// Limit 限制返回的论文数量
	Limit int `json:"limit,omitempty" jsonschema:"description=Maximum number of papers to crawl"`

	// Offset 偏移量，用于分页
	Offset int `json:"offset,omitempty" jsonschema:"description=Offset for pagination"`

	// VenueId OpenReview 平台专用参数
	VenueId string `json:"venue_id,omitempty" jsonschema:"description=Venue ID for OpenReview platform"`
}

type CrawlerOutput struct {
	Count   int    `json:"count" jsonschema:"description=Number of papers successfully crawled"`
	Message string `json:"message" jsonschema:"description=Result message"`
}

// NewCrawlerTool 创建爬虫工具，接受 App 实例
func NewCrawlerTool(app *App) tool.InvokableTool {
	crawlerTool, err := utils.InferTool("crawler", "Crawl academic papers from various platforms (arxiv, openreview, acl, etc.) based on keywords, categories, and date range", func(ctx context.Context, input *CrawlerInput) (output *CrawlerOutput, err error) {
		if app == nil || app.coreApp == nil {
			return nil, fmt.Errorf("app instance is not initialized")
		}

		// 构建 platform.Query
		query := platform.Query{
			Keywords:   input.Keywords,
			Categories: input.Categories,
			DateFrom:   input.DateFrom,
			DateTo:     input.DateTo,
			Limit:      input.Limit,
			Offset:     input.Offset,
		}

		if input.Platform == "openreview" && input.VenueId != "" {
			query.Categories = []string{input.VenueId}
		}

		count, err := app.coreApp.Crawl(ctx, input.Platform, query)
		if err != nil {
			return &CrawlerOutput{
				Count:   0,
				Message: fmt.Sprintf("Crawl failed: %v", err),
			}, err
		}

		return &CrawlerOutput{
			Count:   count,
			Message: fmt.Sprintf("Successfully crawled %d papers from %s", count, input.Platform),
		}, nil
	})

	if err != nil {
		log.Fatalf("failed to create crawler tool: %v", err)
	}

	return crawlerTool
}
