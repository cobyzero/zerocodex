package domain

type ProjectRepository interface {
	Validate(path string) bool
	ListFiles(path string) string
	ReadFile(basePath, relPath string) (string, error)
	WriteFile(basePath, relPath, content string) error
}

type LLMClient interface {
	Chat(
		systemContext,
		prompt string,
		readFunc func(string) string,
		writeFunc func(path, content string) string,
		runCommandFunc func(command string) string,
	) (string, error)
}
