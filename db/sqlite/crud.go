package db

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"sort"
	"strings"

	"PaperHunter/internal/models"
	"PaperHunter/pkg/similarity"

	_ "github.com/mattn/go-sqlite3"
)

// 总逻辑：输入关键词 -> embedding 生成向量, 然后存储层保存向量到内存 -> 查询数据库中已经生成向量的对应关系
// 爬取时也需要将对应的标题 embedding

func (s *SQLiteDB) Upsert(p *models.Paper) (int64, error) {
	query := `
	INSERT INTO papers (
		source, source_id, url, title, title_translated,
		authors, abstract, abstract_translated, categories, comments,
		first_submitted_at, first_announced_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(source, source_id) DO UPDATE SET
		title = excluded.title,
		title_translated = excluded.title_translated,
		authors = excluded.authors,
		abstract = excluded.abstract,
		abstract_translated = excluded.abstract_translated,
		categories = excluded.categories,
		comments = excluded.comments,
		first_submitted_at = excluded.first_submitted_at,
		first_announced_at = excluded.first_announced_at,
		updated_at = CURRENT_TIMESTAMP
	RETURNING id
	`

	var id int64
	err := s.db.QueryRow(query,
		p.Source, p.SourceID, p.URL, p.Title, p.TitleTranslated,
		p.AuthorsCSV(), p.Abstract, p.AbstractTranslated,
		p.CategoriesCSV(), p.Comments,
		p.FirstSubmittedAt, p.FirstAnnouncedAt,
	).Scan(&id)

	return id, err
}

// SaveEmbedding 保存论文的向量表示
func (s *SQLiteDB) SaveEmbedding(paperID int64, model, text string, vec []float32) error {
	blob := encodeVec(vec)
	query := `
	UPDATE papers SET 
		embedding_text = ?,
		embedding = ?,
		embedding_model = ?,
		embedding_updated_at = CURRENT_TIMESTAMP
	WHERE id = ?
	`

	_, err := s.db.Exec(query, text, blob, model, paperID)
	return err
}

// GetPapersNeedingEmbedding 获取需要计算向量的论文
func (s *SQLiteDB) GetPapersNeedingEmbedding(model string, limit int) ([]*models.Paper, error) {
	query := `
	SELECT id, source, source_id, url, title, title_translated, authors,
		abstract, abstract_translated, categories, comments,
		first_submitted_at, first_announced_at, updated_at
	FROM papers 
	WHERE embedding IS NULL OR embedding_model != ?
	LIMIT ?
	`

	rows, err := s.db.Query(query, model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPapers(rows)
}

// SearchByEmbedding 基于向量相似度检索论文
func (s *SQLiteDB) SearchByEmbedding(queryVec []float32, model string, cond models.SearchCondition, topK int) ([]*models.SimilarPaper, error) {

	where := []string{"embedding IS NOT NULL", "embedding_model = ?"}
	args := []interface{}{model}

	if len(cond.Sources) > 0 {
		placeholders := strings.Repeat("?,", len(cond.Sources))
		placeholders = placeholders[:len(placeholders)-1]
		where = append(where, "source IN ("+placeholders+")")
		for _, src := range cond.Sources {
			args = append(args, src)
		}
	}

	if cond.DateFrom != nil {
		where = append(where, "first_announced_at >= ?")
		args = append(args, *cond.DateFrom)
	}

	if cond.DateTo != nil {
		where = append(where, "first_announced_at <= ?")
		args = append(args, *cond.DateTo)
	}

	query := `
	SELECT id, source, source_id, url, title, title_translated, authors,
		abstract, abstract_translated, categories, comments,
		first_submitted_at, first_announced_at, updated_at, embedding
	FROM papers 
	WHERE ` + strings.Join(where, " AND ")

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.SimilarPaper
	for rows.Next() {
		var p models.Paper
		var authorsStr, categoriesStr string
		var embBlob []byte

		err := rows.Scan(
			&p.ID, &p.Source, &p.SourceID, &p.URL, &p.Title, &p.TitleTranslated,
			&authorsStr, &p.Abstract, &p.AbstractTranslated, &categoriesStr, &p.Comments,
			&p.FirstSubmittedAt, &p.FirstAnnouncedAt, &p.UpdatedAt, &embBlob,
		)
		if err != nil {
			return nil, err
		}

		if authorsStr != "" {
			p.Authors = strings.Split(strings.Trim(authorsStr, ","), ",")
		}
		if categoriesStr != "" {
			p.Categories = strings.Split(strings.Trim(categoriesStr, ","), ",")
		}

		vec := decodeVec(embBlob)
		sim := similarity.CosineSimilarity(queryVec, vec)

		results = append(results, &models.SimilarPaper{
			Paper:      p,
			Similarity: sim,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > topK {
		results = results[:topK]
	}

	return results, rows.Err()
}

func (s *SQLiteDB) scanPapers(rows *sql.Rows) ([]*models.Paper, error) {
	var papers []*models.Paper

	for rows.Next() {
		var p models.Paper
		var authorsStr, categoriesStr string

		err := rows.Scan(
			&p.ID, &p.Source, &p.SourceID, &p.URL, &p.Title, &p.TitleTranslated,
			&authorsStr, &p.Abstract, &p.AbstractTranslated, &categoriesStr, &p.Comments,
			&p.FirstSubmittedAt, &p.FirstAnnouncedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if authorsStr != "" {
			p.Authors = strings.Split(strings.Trim(authorsStr, ","), ",")
		}
		if categoriesStr != "" {
			p.Categories = strings.Split(strings.Trim(categoriesStr, ","), ",")
		}

		papers = append(papers, &p)
	}

	return papers, rows.Err()
}

func encodeVec(vec []float32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, vec)
	return buf.Bytes()
}

func decodeVec(blob []byte) []float32 {
	vec := make([]float32, len(blob)/4)
	buf := bytes.NewReader(blob)
	_ = binary.Read(buf, binary.LittleEndian, &vec)
	return vec
}

func (s *SQLiteDB) CountPapers(conditions []string, params []interface{}) (int, error) {
	query := "SELECT COUNT(*) FROM papers"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := s.db.QueryRow(query, params...).Scan(&count)
	return count, err
}

func (s *SQLiteDB) DeletePapers(conditions []string, params []interface{}) (int, error) {
	query := "DELETE FROM papers"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	result, err := s.db.Exec(query, params...)
	if err != nil {
		return 0, err
	}

	count, err := result.RowsAffected()
	return int(count), err
}

func (s *SQLiteDB) SearchByKeywords(query string, cond models.SearchCondition) ([]*models.Paper, error) {

	where := []string{"(title LIKE ? OR abstract LIKE ?)"}
	searchPattern := "%" + query + "%"
	args := []interface{}{searchPattern, searchPattern}

	if len(cond.Sources) > 0 {
		placeholders := strings.Repeat("?,", len(cond.Sources))
		placeholders = placeholders[:len(placeholders)-1]
		where = append(where, "source IN ("+placeholders+")")
		for _, src := range cond.Sources {
			args = append(args, src)
		}
	}

	if cond.DateFrom != nil {
		where = append(where, "first_announced_at >= ?")
		args = append(args, *cond.DateFrom)
	}

	if cond.DateTo != nil {
		where = append(where, "first_announced_at <= ?")
		args = append(args, *cond.DateTo)
	}

	sqlQuery := `
	SELECT id, source, source_id, url, title, title_translated, authors,
		abstract, abstract_translated, categories, comments,
		first_submitted_at, first_announced_at, updated_at
	FROM papers 
	WHERE ` + strings.Join(where, " AND ")

	if cond.Limit > 0 {
		sqlQuery += " LIMIT ?"
		args = append(args, cond.Limit)
	}

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPapers(rows)
}

func (s *SQLiteDB) GetPapersByConditions(conditions []string, params []interface{}, limit int) ([]*models.Paper, error) {
	query := `
	SELECT id, source, source_id, url, title, title_translated, authors,
		abstract, abstract_translated, categories, comments,
		first_submitted_at, first_announced_at, updated_at
	FROM papers`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// 添加 LIMIT 子句
	if limit > 0 {
		query += " LIMIT ?"
		params = append(params, limit)
	}

	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPapers(rows)
}

func (s *SQLiteDB) GetPapersList(limit, offset int, conditions []string, params []interface{}, orderBy string) ([]*models.Paper, int, error) {
	//计算总量
	countQuery := "SELECT COUNT(*) FROM papers"
	if len(conditions) > 0 {
		countQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	err := s.db.QueryRow(countQuery, params...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 直接查询即可
	query := `
	SELECT id, source, source_id, url, title, title_translated, authors,
		abstract, abstract_translated, categories, comments,
		first_submitted_at, first_announced_at, updated_at
	FROM papers`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if orderBy != "" {
		query += " ORDER BY " + orderBy
	} else {
		query += " ORDER BY first_announced_at DESC"
	}

	query += " LIMIT ? OFFSET ?"
	params = append(params, limit, offset)

	rows, err := s.db.Query(query, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	papers, err := s.scanPapers(rows)
	return papers, total, err
}
