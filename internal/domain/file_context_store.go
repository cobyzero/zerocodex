package domain

type FileContextRecord struct {
	FilePath string
	Size     int64
	ModUnix  int64
	Context  string
}

type FileContextStore interface {
	ListByProject(projectPath string) ([]FileContextRecord, error)
	Upsert(projectPath, filePath string, size, modUnix int64, context string) error
}
