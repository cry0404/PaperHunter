package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ExportInput 导出工具的输入参数
type ExportInput struct {
	// Format 导出格式：csv, json, zotero, feishu
	Format string `json:"format" jsonschema:"required,enum=csv,enum=json,enum=zotero,enum=feishu,description=Export format (csv, json, zotero, feishu)"`

	// Output 输出文件路径（csv/json 格式必填）
	Output string `json:"output,omitempty" jsonschema:"description=Output file path (required for csv/json format)"`

	// Query 查询字符串过滤（在标题或摘要中搜索）
	Query string `json:"query,omitempty" jsonschema:"description=Filter by query string (searches in title or abstract)"`

	// Keywords 关键词列表（多个关键词用 AND 连接）
	Keywords []string `json:"keywords,omitempty" jsonschema:"description=List of keywords to filter (AND logic)"`

	// Categories 类别列表（多个类别用 OR 连接）
	Categories []string `json:"categories,omitempty" jsonschema:"description=List of categories to filter (OR logic)"`

	// Source 数据源过滤（如 arxiv, openreview, acl 等）
	Source string `json:"source,omitempty" jsonschema:"description=Filter by data source (e.g., arxiv, openreview, acl)"`

	// Collection Zotero 集合 key（用于 zotero 格式）
	Collection string `json:"collection,omitempty" jsonschema:"description=Zotero collection key (for zotero format)"`

	// FeishuName 飞书多维表格名称（用于 feishu 格式）
	FeishuName string `json:"feishu_name,omitempty" jsonschema:"description=Feishu Bitable name (for feishu format)"`

	// Limit 导出数量限制（0 表示不限制）
	Limit int `json:"limit,omitempty" jsonschema:"description=Export limit (0 means no limit)"`
}

// ExportOutput 导出工具的输出结果
type ExportOutput struct {
	Success bool   `json:"success" jsonschema:"description=Whether the export was successful"`
	Message string `json:"message" jsonschema:"description=Result message"`
	URL     string `json:"url,omitempty" jsonschema:"description=Export URL (for feishu format)"`
}

// NewExportTool 创建导出工具，接受 App 实例
func NewExportTool(app *App) tool.InvokableTool {
	exportTool, err := utils.InferTool("export", "Export papers to different formats (csv, json, zotero, feishu) with optional filtering", func(ctx context.Context, input *ExportInput) (output *ExportOutput, err error) {
		if app == nil || app.coreApp == nil {
			return nil, fmt.Errorf("app instance is not initialized")
		}

		validFormats := map[string]bool{"csv": true, "json": true, "zotero": true, "feishu": true}
		if !validFormats[strings.ToLower(input.Format)] {
			return &ExportOutput{
				Success: false,
				Message: fmt.Sprintf("Unsupported format: %s. Supported formats: csv, json, zotero, feishu", input.Format),
			}, fmt.Errorf("unsupported format: %s", input.Format)
		}

		if (input.Format == "csv" || input.Format == "json") && strings.TrimSpace(input.Output) == "" {
			return &ExportOutput{
				Success: false,
				Message: "Output path is required for csv/json format",
			}, fmt.Errorf("output path is required for csv/json format")
		}

		var conditions []string
		var params []interface{}

		if input.Source != "" {
			conditions = append(conditions, "source = ?")
			params = append(params, input.Source)
		}

		if input.Query != "" {
			conditions = append(conditions, "(title LIKE ? OR abstract LIKE ?)")
			pattern := "%" + input.Query + "%"
			params = append(params, pattern, pattern)
		}

		if len(input.Keywords) > 0 {
			keywordConds := make([]string, 0, len(input.Keywords))
			for _, keyword := range input.Keywords {
				keywordConds = append(keywordConds, "(title LIKE ? OR abstract LIKE ?)")
				pattern := "%" + keyword + "%"
				params = append(params, pattern, pattern)
			}
			if len(keywordConds) > 0 {
				conditions = append(conditions, "("+strings.Join(keywordConds, " AND ")+")")
			}
		}

		if len(input.Categories) > 0 {
			catConds := make([]string, 0, len(input.Categories))
			for _, cat := range input.Categories {
				catConds = append(catConds, "categories LIKE ?")
				params = append(params, "%"+cat+"%")
			}
			if len(catConds) > 0 {
				conditions = append(conditions, "("+strings.Join(catConds, " OR ")+")")
			}
		}

		switch strings.ToLower(input.Format) {
		case "csv", "json":
			err := app.coreApp.ExportPapers(ctx, input.Format, input.Output, conditions, params, input.Limit)
			if err != nil {
				return &ExportOutput{
					Success: false,
					Message: fmt.Sprintf("Export failed: %v", err),
				}, err
			}
			return &ExportOutput{
				Success: true,
				Message: fmt.Sprintf("Successfully exported to %s", input.Output),
			}, nil

		case "zotero":
			err := app.coreApp.ExportToZotero(ctx, input.Collection, conditions, params, input.Limit)
			if err != nil {
				return &ExportOutput{
					Success: false,
					Message: fmt.Sprintf("Export to Zotero failed: %v", err),
				}, err
			}
			return &ExportOutput{
				Success: true,
				Message: "Successfully exported to Zotero",
			}, nil

		case "feishu":
			name := strings.TrimSpace(input.FeishuName)
			if name == "" {
				return &ExportOutput{
					Success: false,
					Message: "FeishuName is required for feishu format",
				}, fmt.Errorf("feishu_name is required for feishu format")
			}
			url, err := app.coreApp.ExportToFeiShuBitableWithURL(ctx, name, name, conditions, params, input.Limit)
			if err != nil {
				return &ExportOutput{
					Success: false,
					Message: fmt.Sprintf("Export to Feishu failed: %v", err),
				}, err
			}
			return &ExportOutput{
				Success: true,
				Message: "Successfully exported to Feishu",
				URL:     url,
			}, nil

		default:
			return &ExportOutput{
				Success: false,
				Message: fmt.Sprintf("Unknown format: %s", input.Format),
			}, fmt.Errorf("unknown format: %s", input.Format)
		}
	})

	if err != nil {
		log.Fatalf("failed to create export tool: %v", err)
	}

	return exportTool
}

