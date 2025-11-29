package main

import (
	"context"
	"fmt"
	"time"
)


type CleanOptions struct {
	Source       string `json:"source"`
	From         string `json:"from"`  // YYYY-MM-DD
	Until        string `json:"until"` // YYYY-MM-DD
	WithoutEmbed bool   `json:"withoutEmbed"`
	ExportBefore bool   `json:"exportBefore"`
	ExportFormat string `json:"exportFormat"` // csv|json
	ExportOutput string `json:"exportOutput"`
}

type CleanResult struct {
	Matched      int    `json:"matched"`
	Deleted      int    `json:"deleted"`
	ExportedPath string `json:"exportedPath,omitempty"`
}

// CleanWithOptions 统计、可选导出并清理
func (a *App) CleanWithOptions(opts CleanOptions) (CleanResult, error) {
	if a.coreApp == nil {
		return CleanResult{}, fmt.Errorf("app not initialized")
	}

	var conditions []string
	var params []interface{}

	if opts.Source != "" {
		conditions = append(conditions, "source = ?")
		params = append(params, opts.Source)
	}
	if opts.From != "" {
		t, err := time.Parse("2006-01-02", opts.From)
		if err != nil {
			return CleanResult{}, fmt.Errorf("invalid from date: %w", err)
		}
		conditions = append(conditions, "first_announced_at >= ?")
		params = append(params, t)
	}
	if opts.Until != "" {
		t, err := time.Parse("2006-01-02", opts.Until)
		if err != nil {
			return CleanResult{}, fmt.Errorf("invalid until date: %w", err)
		}
		conditions = append(conditions, "first_announced_at <= ?")
		params = append(params, t)
	}
	if opts.WithoutEmbed {
		conditions = append(conditions, "embedding IS NULL")
	}

	if len(conditions) == 0 {
		return CleanResult{}, fmt.Errorf("no conditions provided")
	}

	ctx := context.Background()

	matched, err := a.coreApp.CountPapers(ctx, conditions, params)
	if err != nil {
		return CleanResult{}, err
	}
	if matched == 0 {
		return CleanResult{Matched: 0, Deleted: 0}, nil
	}

	res := CleanResult{Matched: matched}

	if opts.ExportBefore {
		format := opts.ExportFormat
		if format != "csv" && format != "json" {
			return CleanResult{}, fmt.Errorf("invalid export format: %s", format)
		}
		output := opts.ExportOutput
		if output == "" {
			output = fmt.Sprintf("papers_export_%s.%s", time.Now().Format("20060102_150405"), format)
		}
		if err := a.coreApp.ExportPapers(ctx, format, output, conditions, params, matched); err != nil {
			return CleanResult{}, err
		}
		res.ExportedPath = output
	}

	deleted, err := a.coreApp.DeletePapers(ctx, conditions, params)
	if err != nil {
		return CleanResult{}, err
	}
	res.Deleted = deleted
	return res, nil
}

