package export

import (
	"PaperHunter/internal/models"
)

// Exporter 导出器接口
type Exporter interface {
	// Export 导出论文到指定文件
	Export(papers []*models.Paper, outputPath string) error
}
