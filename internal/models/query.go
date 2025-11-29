package models



import (
	"time"
)

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