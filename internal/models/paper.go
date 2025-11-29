package models

import (
	"strings"
	"time"
)




type SimilarPaper struct {
    Paper      Paper
    Similarity float32 //与关键词的匹配相似度，这里主要是定义相似度多少就可以存储
}

// Paper 统一的论文数据模型，独立于具体平台（arXiv/ACL 等）

type Paper struct {
	ID                int64     `db:"id"`
	Source            string    `db:"source"`     // 平台标识，如: "arxiv", "acl", "dblp", "semantic"
	SourceID          string    `db:"source_id"` // 平台内唯一ID，如: arXivID
	URL               string    `db:"url"`
	Title             string    `db:"title"`
	TitleTranslated   string    `db:"title_translated"`
	Authors           []string  `db:"-"`
	Abstract          string    `db:"abstract"`
	AbstractTranslated string   `db:"abstract_translated"`
	Categories        []string  `db:"-"`
	Comments          string    `db:"comments"`
	FirstSubmittedAt  time.Time `db:"first_submitted_date"`
	FirstAnnouncedAt  time.Time `db:"first_announced_date"`
	UpdatedAt         time.Time `db:"update_time"`
}

// AuthorsCSV 返回以逗号分隔的作者名
func (p *Paper) AuthorsCSV() string {
	return strings.Join(p.Authors, ", ")
}

// CategoriesCSV 返回以逗号分隔的类别
func (p *Paper) CategoriesCSV() string {
	return strings.Join(p.Categories, ", ")
}
