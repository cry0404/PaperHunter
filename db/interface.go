package db

import (
	"PaperHunter/internal/models"
)

// 为 postgreSql 保留一下接口, 理论上命令行程序应该简洁更好, 但万一发了呢
type PaperStorage interface {
	Upsert(paper *models.Paper) (int64, error)

	SaveEmbedding(paperID int64, model string, text string, vec []float32) error

	GetPapersNeedingEmbedding(model string, limit int) ([]*models.Paper, error)

	SearchByEmbedding(queryVec []float32, model string, cond models.SearchCondition, topK int) ([]*models.SimilarPaper, error)

	SearchByKeywords(query string, cond models.SearchCondition) ([]*models.Paper, error)

	CountPapers(conditions []string, params []interface{}) (int, error)

	DeletePapers(conditions []string, params []interface{}) (int, error)

	GetPapersByConditions(conditions []string, params []interface{}, limit int) ([]*models.Paper, error)

	GetPapersList(limit, offset int, conditions []string, params []interface{}, orderBy string) ([]*models.Paper, int, error)

	Close() error
}
