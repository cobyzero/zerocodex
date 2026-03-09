package filesystem

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProjectFS struct{}

const maxListedFiles = 500

var ignoredDirs = map[string]struct{}{
	".git":       {},
	".idea":      {},
	".vscode":    {},
	"node_modules": {},
	"dist":       {},
	"build":      {},
	"target":     {},
	".venv":      {},
	"venv":       {},
	"__pycache__": {},
	"vendor":     {},
}

func (p *ProjectFS) Validate(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (p *ProjectFS) ListFiles(path string) string {
	files := make([]string, 0, maxListedFiles)
	_ = filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if _, ignored := ignoredDirs[info.Name()]; ignored {
				return filepath.SkipDir
			}
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if len(files) >= maxListedFiles {
			return nil
		}
		relPath, _ := filepath.Rel(path, fp)
		files = append(files, relPath)
		return nil
	})

	sort.Strings(files)

	var builder strings.Builder
	for _, relPath := range files {
		builder.WriteString("- " + relPath + "\n")
	}
	if len(files) >= maxListedFiles {
		builder.WriteString("... [list truncated]\n")
	}
	return builder.String()
}

func (p *ProjectFS) ReadFile(basePath, relPath string) (string, error) {
	fullPath := filepath.Join(basePath, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (p *ProjectFS) WriteFile(basePath, relPath, content string) error {
	fullPath := filepath.Join(basePath, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0o644)
}
