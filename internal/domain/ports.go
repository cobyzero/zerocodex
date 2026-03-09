package domain

type ProjectRepository interface {
	Validate(path string) bool
	ListFiles(path string) string
	ReadFile(basePath, relPath string) (string, error)
}

type LLMClient interface {
	Chat(systemContext, prompt string, readFunc func(string) string) (string, error)
}
