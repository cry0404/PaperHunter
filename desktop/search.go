package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"PaperHunter/internal/core"
	"PaperHunter/internal/models"
)

// SearchExample 定义在 searchTool.go 中

type SearchOptions struct {
	Query        string          `json:"query"`
	Examples     []SearchExample `json:"examples"`
	Semantic     bool            `json:"semantic"`
	TopK         int             `json:"topK"`
	Limit        int             `json:"limit"`
	Source       string          `json:"source"`
	From         string          `json:"from"`  // YYYY-MM-DD
	Until        string          `json:"until"` // YYYY-MM-DD
	ComputeEmbed bool            `json:"computeEmbed"`
	EmbedBatch   int             `json:"embedBatch"`
}

// SearchWithOptions 执行搜索并返回 JSON 字符串结果
func (a *App) SearchWithOptions(opts SearchOptions) (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("app not initialized")
	}

	ctx := context.Background()

	if opts.ComputeEmbed {
		batch := opts.EmbedBatch
		if batch <= 0 {
			batch = 100
		}
		if _, err := a.coreApp.ComputeMissingEmbeddings(ctx, batch); err != nil {
			return "", fmt.Errorf("compute embeddings failed: %w", err)
		}
	}

	cond := models.SearchCondition{Limit: opts.Limit}

	if opts.Source != "" {
		cond.Sources = []string{opts.Source}
	}

	if opts.From != "" {
		t, err := time.Parse("2006-01-02", opts.From)
		if err != nil {
			return "", fmt.Errorf("invalid from date: %w", err)
		}
		cond.DateFrom = &t
	}

	if opts.Until != "" {
		t, err := time.Parse("2006-01-02", opts.Until)
		if err != nil {
			return "", fmt.Errorf("invalid until date: %w", err)
		}
		cond.DateTo = &t
	}

	// 转换示例
	var examples []*models.Paper
	if len(opts.Examples) > 0 {
		for _, e := range opts.Examples {
			p := &models.Paper{Title: e.Title, Abstract: e.Abstract}
			if p.Title != "" || p.Abstract != "" {
				examples = append(examples, p)
			}
		}
	}

	sopts := core.SearchOptions{
		Query:     opts.Query,
		Examples:  examples,
		Condition: cond,
		TopK:      opts.TopK,
		Semantic:  opts.Semantic,
	}

	results, err := a.coreApp.Search(ctx, sopts)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
