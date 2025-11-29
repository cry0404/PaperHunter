package main

import (
	"context"
	"fmt"
	"strings"
)

type ExportOptions struct {
	Format     string   `json:"format"` // csv|json|zotero|feishu
	Output     string   `json:"output"` // csv/json 必填
	Query      string   `json:"query"`
	Keywords   []string `json:"keywords"`
	Categories []string `json:"categories"`
	Source     string   `json:"source"`
	Collection string   `json:"collection"` // zotero
	FeishuName string   `json:"feishuName"` // feishu: 作为文件与文件夹名
	Limit      int      `json:"limit"`
}

func (a *App) ExportWithOptions(opts ExportOptions) (string, error) {
	if a.coreApp == nil {
		return "", fmt.Errorf("app not initialized")
	}

	valid := map[string]bool{"csv": true, "json": true, "zotero": true, "feishu": true}
	if !valid[strings.ToLower(opts.Format)] {
		return "", fmt.Errorf("unsupported format: %s", opts.Format)
	}

	// csv/json 必须提供输出
	if (opts.Format == "csv" || opts.Format == "json") && strings.TrimSpace(opts.Output) == "" {
		return "", fmt.Errorf("output is required for csv/json")
	}

	// 组装 conditions/params
	var conditions []string
	var params []interface{}

	if opts.Source != "" {
		conditions = append(conditions, "source = ?")
		params = append(params, opts.Source)
	}
	if opts.Query != "" {
		conditions = append(conditions, "(title LIKE ? OR abstract LIKE ?)")
		pattern := "%" + opts.Query + "%"
		params = append(params, pattern, pattern)
	}
	if len(opts.Keywords) > 0 {
		ks := make([]string, 0, len(opts.Keywords))
		for range opts.Keywords {
			ks = append(ks, "(title LIKE ? OR abstract LIKE ?)")
		}
		conditions = append(conditions, "("+strings.Join(ks, " AND ")+")")
		for _, k := range opts.Keywords {
			p := "%" + k + "%"
			params = append(params, p, p)
		}
	}
	if len(opts.Categories) > 0 {
		cs := make([]string, 0, len(opts.Categories))
		for range opts.Categories {
			cs = append(cs, "categories LIKE ?")
		}
		conditions = append(conditions, "("+strings.Join(cs, " OR ")+")")
		for _, c := range opts.Categories {
			params = append(params, "%"+c+"%")
		}
	}

	ctx := context.Background()

	switch opts.Format {
	case "csv", "json":
		return opts.Output, a.coreApp.ExportPapers(ctx, opts.Format, opts.Output, conditions, params, opts.Limit)
	case "zotero":
		return "", a.coreApp.ExportToZotero(ctx, opts.Collection, conditions, params, opts.Limit)
	case "feishu":
		name := strings.TrimSpace(opts.FeishuName)
		if name == "" {
			return "", fmt.Errorf("feishuName is required for feishu export")
		}
		url, err := a.coreApp.ExportToFeiShuBitableWithURL(ctx, name, name, conditions, params, opts.Limit)
		if err != nil {
			return "", err
		}
		fmt.Println("Feishu URL:", url)
		return url, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", opts.Format)
	}
}
