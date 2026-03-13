package domain

type ChatHistoryEntry struct {
	Role    string
	Content string
}

type ChatHistoryStore interface {
	Append(projectPath, role, content string) error
	ListRecent(projectPath string, limit int) ([]ChatHistoryEntry, error)
	Clear(projectPath string) error
}
