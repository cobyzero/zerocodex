package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type ProjectFS struct {
	mu        sync.RWMutex
	fileCache map[string]cachedFile
	listCache map[string]cachedList
}

type cachedFile struct {
	modUnix int64
	size    int64
	content string
}

type cachedList struct {
	modUnix int64
	content string
}

const maxListedFiles = 500

var ignoredDirs = map[string]struct{}{
	".git":         {},
	".idea":        {},
	".vscode":      {},
	"node_modules": {},
	"dist":         {},
	"build":        {},
	"target":       {},
	".venv":        {},
	"venv":         {},
	"__pycache__":  {},
	"vendor":       {},
}

func (p *ProjectFS) ensureCaches() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.fileCache == nil {
		p.fileCache = map[string]cachedFile{}
	}
	if p.listCache == nil {
		p.listCache = map[string]cachedList{}
	}
}

func (p *ProjectFS) Validate(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (p *ProjectFS) ListFiles(path string) string {
	p.ensureCaches()
	if content, ok := p.getCachedList(path); ok {
		return content
	}

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
	content := builder.String()
	p.setCachedList(path, content)
	return content
}

func (p *ProjectFS) ReadFile(basePath, relPath string) (string, error) {
	p.ensureCaches()
	fullPath := filepath.Join(basePath, relPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", err
	}
	if content, ok := p.getCachedFile(fullPath, info.ModTime().Unix(), info.Size()); ok {
		return content, nil
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	text := string(content)
	p.setCachedFile(fullPath, info.ModTime().Unix(), info.Size(), text)
	return text, nil
}

func (p *ProjectFS) WriteFile(basePath, relPath, content string) error {
	p.ensureCaches()
	fullPath := filepath.Join(basePath, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return err
	}

	info, err := os.Stat(fullPath)
	if err == nil {
		p.setCachedFile(fullPath, info.ModTime().Unix(), info.Size(), content)
	}
	p.invalidateListCache(basePath)
	return nil
}

func (p *ProjectFS) ReadFileRange(basePath, relPath string, startLine, endLine int) (string, error) {
	if startLine < 1 || endLine < startLine {
		return "", fmt.Errorf("invalid line range")
	}

	fullPath := filepath.Join(basePath, relPath)
	file, err := os.Open(fullPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Allow reasonably large single lines without falling back to full file reads.
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	lineNo := 0
	var out strings.Builder
	for scanner.Scan() {
		lineNo++
		if lineNo < startLine {
			continue
		}
		if lineNo > endLine {
			break
		}
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if lineNo < startLine {
		return "", fmt.Errorf("requested range L%d-L%d is out of bounds. file has %d lines", startLine, endLine, lineNo)
	}
	return out.String(), nil
}

func (p *ProjectFS) getCachedFile(fullPath string, modUnix, size int64) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	entry, ok := p.fileCache[fullPath]
	if !ok || entry.modUnix != modUnix || entry.size != size {
		return "", false
	}
	return entry.content, true
}

func (p *ProjectFS) setCachedFile(fullPath string, modUnix, size int64, content string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fileCache[fullPath] = cachedFile{
		modUnix: modUnix,
		size:    size,
		content: content,
	}
}

func (p *ProjectFS) getCachedList(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	entry, ok := p.listCache[path]
	if !ok || entry.modUnix != info.ModTime().Unix() {
		return "", false
	}
	return entry.content, true
}

func (p *ProjectFS) setCachedList(path, content string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.listCache[path] = cachedList{
		modUnix: info.ModTime().Unix(),
		content: content,
	}
}

func (p *ProjectFS) invalidateListCache(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.listCache, path)
}
