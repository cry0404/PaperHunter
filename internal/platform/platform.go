package platform

import (
	"context"

	"PaperHunter/internal/models"
)

// Query 平台查询参数（统一接口）, 这里是针对爬虫的逻辑, 也是 cli 对应配置
type Query struct {
	Keywords   []string
	Categories []string
	DateFrom   string // YYYY-MM-DD
	DateTo     string // YYYY-MM-DD
	Limit      int
	Offset     int
}

// Result 查询结果
type Result struct {
	Total  int
	Papers []*models.Paper
}

// Platform 平台接口，所有平台（arXiv/ACL/DBLP/Semantic）都需实现
type Platform interface {
	Name() string

	// Search 执行搜索查询, 这里是实现爬取
	Search(ctx context.Context, q Query) (Result, error)

	GetConfig() Config
}

type Config interface {
	Validate() error
}
