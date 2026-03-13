package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type SQLiteProjectStore struct {
	db *sql.DB
}

func NewSQLiteProjectStore(dbPath string) (*SQLiteProjectStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteProjectStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteProjectStore) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS projects (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL UNIQUE,
  last_used_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`)
	return err
}

func (s *SQLiteProjectStore) SaveProject(path string) error {
	_, err := s.db.Exec(`
INSERT INTO projects(path, last_used_at)
VALUES(?, CURRENT_TIMESTAMP)
ON CONFLICT(path) DO UPDATE SET last_used_at = CURRENT_TIMESTAMP;
`, path)
	return err
}

func (s *SQLiteProjectStore) ListProjects() ([]string, error) {
	rows, err := s.db.Query(`
SELECT path
FROM projects
ORDER BY last_used_at DESC;
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := []string{}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		projects = append(projects, path)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *SQLiteProjectStore) RemoveProject(path string) error {
	_, err := s.db.Exec(`
DELETE FROM projects
WHERE path = ?;
`, path)
	return err
}

func (s *SQLiteProjectStore) Close() error {
	return s.db.Close()
}
