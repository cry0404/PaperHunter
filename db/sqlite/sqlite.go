package db

//应该根据不同的 source 创建不同的表？
//❌可以通过导出表格筛选
import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(path string) (*SQLiteDB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建目录，请检查权限问题: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库，请检查权限问题: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	sqlDB := &SQLiteDB{db: db}

	if err := sqlDB.initTable(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("数据库创建失败: %w", err)
	}

	return sqlDB, nil
}

func (d *SQLiteDB) Close() error { return d.db.Close() }

func (d *SQLiteDB) initTable() error {
	schema := `
CREATE TABLE IF NOT EXISTS papers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source TEXT NOT NULL,
  source_id TEXT NOT NULL,
  url TEXT UNIQUE NOT NULL,
  title TEXT NOT NULL,
  title_translated TEXT,
  authors TEXT,                  -- 存 ",a1,a2," 便于 LIKE 精确匹配
  abstract TEXT,
  abstract_translated TEXT,
  categories TEXT,               -- 存 ",cs.AI,cs.LG,"
  comments TEXT,
  first_submitted_at DATETIME,
  first_announced_at DATETIME,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

  -- 向量相关
  embedding_text TEXT,           -- 生成向量用的原始文本（title+abstract 等）
  embedding BLOB,                -- float32 数组（二进制）
  embedding_model TEXT,
  embedding_updated_at DATETIME,

  UNIQUE(source, source_id)
);

CREATE INDEX IF NOT EXISTS idx_papers_source ON papers(source);
CREATE INDEX IF NOT EXISTS idx_papers_date ON papers(first_announced_at);
CREATE INDEX IF NOT EXISTS idx_papers_model ON papers(embedding_model);  

	`

	_, err := d.db.Exec(schema)

	return err
}
