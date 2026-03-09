package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/cobyzero/zerocodex/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteFileContextStore struct {
	db *sql.DB
}

func NewSQLiteFileContextStore(dbPath string) (*SQLiteFileContextStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteFileContextStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteFileContextStore) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS file_contexts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_path TEXT NOT NULL,
  file_path TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  mod_unix INTEGER NOT NULL,
  context TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(project_path, file_path)
);
`)
	return err
}

func (s *SQLiteFileContextStore) ListByProject(projectPath string) ([]domain.FileContextRecord, error) {
	rows, err := s.db.Query(`
SELECT file_path, size_bytes, mod_unix, context
FROM file_contexts
WHERE project_path = ?
ORDER BY file_path ASC;
`, projectPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []domain.FileContextRecord{}
	for rows.Next() {
		var r domain.FileContextRecord
		if err := rows.Scan(&r.FilePath, &r.Size, &r.ModUnix, &r.Context); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *SQLiteFileContextStore) Upsert(projectPath, filePath string, size, modUnix int64, context string) error {
	_, err := s.db.Exec(`
INSERT INTO file_contexts(project_path, file_path, size_bytes, mod_unix, context, updated_at)
VALUES(?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(project_path, file_path)
DO UPDATE SET
  size_bytes = excluded.size_bytes,
  mod_unix = excluded.mod_unix,
  context = excluded.context,
  updated_at = CURRENT_TIMESTAMP;
`, projectPath, filePath, size, modUnix, context)
	return err
}

func (s *SQLiteFileContextStore) Close() error {
	return s.db.Close()
}
