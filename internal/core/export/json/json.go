package json

import (
	"encoding/json"
	"fmt"
	"os"

	"PaperHunter/internal/models"
)

type JSONExporter struct{}

func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

func (e *JSONExporter) Export(papers []*models.Paper, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")  // 格式化输出
	encoder.SetEscapeHTML(false) // 不转义 HTML 字符

	data := map[string]interface{}{
		"total":  len(papers),
		"papers": papers,
	}

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("写入 JSON 失败: %w", err)
	}

	return nil
}
