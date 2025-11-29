package main

import (
	"context"
	"fmt"

	"PaperHunter/internal/models"
)

type PaperListResponse struct {
	Papers []*models.Paper `json:"papers"`
	Total  int             `json:"total"`
}

// GetPapers 获取论文列表
func (a *App) GetPapers(page int, pageSize int, source string, search string) (*PaperListResponse, error) {
	if a.coreApp == nil {
		return nil, fmt.Errorf("core app not initialized")
	}

	var conditions []string
	var params []interface{}

	if source != "" && source != "all" {
		conditions = append(conditions, "source = ?")
		params = append(params, source)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		conditions = append(conditions, "(title LIKE ? OR abstract LIKE ? OR authors LIKE ?)")
		params = append(params, searchPattern, searchPattern, searchPattern)
	}

	papers, total, err := a.coreApp.GetPapers(context.Background(), page, pageSize, conditions, params, "first_announced_at DESC")
	if err != nil {
		return nil, err
	}

	return &PaperListResponse{
		Papers: papers,
		Total:  total,
	}, nil
}
