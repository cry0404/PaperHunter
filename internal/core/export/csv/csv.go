package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"PaperHunter/internal/models"
)


type CSVExporter struct{}

func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

func (e *CSVExporter) Export(papers []*models.Paper, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	if _, err := file.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("写入 BOM 失败: %w", err)
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头
	headers := []string{
		"ID", "数据源", "平台ID", "标题", "标题译文", "作者",
		"摘要", "摘要译文", "分类", "URL", "首次提交日期", "首次发布日期",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("写入表头失败: %w", err)
	}

	for _, p := range papers {
		record := []string{
			fmt.Sprintf("%d", p.ID),
			p.Source,
			p.SourceID,
			p.Title,
			p.TitleTranslated,
			strings.Join(p.Authors, "; "),
			truncate(p.Abstract, 500),
			truncate(p.AbstractTranslated, 500),
			strings.Join(p.Categories, "; "),
			p.URL,
			formatTime(p.FirstSubmittedAt),
			formatTime(p.FirstAnnouncedAt),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("写入数据失败: %w", err)
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
