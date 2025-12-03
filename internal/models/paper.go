package models

import (
	"strings"
	"time"
)




type SimilarPaper struct {
    Paper      Paper
    Similarity float32 //与关键词的匹配相似度，这里主要是定义相似度多少就可以存储
}


type SearchCondition struct {
    Sources    []string
    Keywords   []string // 走 embedding。可以考虑调用时拼接成向量
    DateFrom   *time.Time
    DateTo     *time.Time
    Limit      int
    Offset     int
}

/*
   "deep learning" → [0.21, 0.15, -0.08, ..., 0.33]  (1536维)

	有三篇论文相关，然后按照相似度排序
    论文 A: "Attention Is All You Need"
           embedding = [0.20, 0.14, -0.09, ..., 0.32]
           相似度 = cosineSimilarity(query, A) = 0.95 ← 很相似
   
   论文 B: "ImageNet Classification with CNNs"
           embedding = [0.18, 0.12, -0.06, ..., 0.30]
           相似度 = 0.82 ← 比较相似
   
   论文 C: "Quantum Computing Basics"
           embedding = [-0.05, 0.30, 0.25, ..., -0.10]
           相似度 = 0.15 ← 不相似

		   
SearchByEmbedding(
    queryVec []float32,  // 查询向量，比如关键词"deep learning"的 embedding
    model string,        // 模型名，如 "text-embedding-3-large"
    cond SearchCondition,// 过滤条件（按平台、日期等）
    topK int            // 返回最相似的前 K 篇
)
*/ 

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


func (p *Paper) AuthorsCSV() string {
	return strings.Join(p.Authors, ", ")
}

func (p *Paper) CategoriesCSV() string {
	return strings.Join(p.Categories, ", ")
}
