package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	"github.com/cobyzero/zerocodex/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteChatHistoryStore struct {
	db *sql.DB
}

func NewSQLiteChatHistoryStore(dbPath string) (*SQLiteChatHistoryStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &SQLiteChatHistoryStore{db: db}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLiteChatHistoryStore) initSchema() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS chat_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  project_path TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_chat_history_project_created
ON chat_history(project_path, created_at, id);
`)
	return err
}

func (s *SQLiteChatHistoryStore) Append(projectPath, role, content string) error {
	_, err := s.db.Exec(`
INSERT INTO chat_history(project_path, role, content)
VALUES(?, ?, ?);
`, projectPath, role, content)
	return err
}

func (s *SQLiteChatHistoryStore) ListRecent(projectPath string, limit int) ([]domain.ChatHistoryEntry, error) {
	rows, err := s.db.Query(`
SELECT role, content
FROM (
  SELECT role, content, created_at, id
  FROM chat_history
  WHERE project_path = ?
  ORDER BY created_at DESC, id DESC
  LIMIT ?
)
ORDER BY created_at ASC, id ASC;
`, projectPath, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []domain.ChatHistoryEntry{}
	for rows.Next() {
		var entry domain.ChatHistoryEntry
		if err := rows.Scan(&entry.Role, &entry.Content); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *SQLiteChatHistoryStore) Clear(projectPath string) error {
	_, err := s.db.Exec(`
DELETE FROM chat_history
WHERE project_path = ?;
`, projectPath)
	return err
}

func (s *SQLiteChatHistoryStore) Close() error {
	return s.db.Close()
}
