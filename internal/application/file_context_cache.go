package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxFilesToIndex        = 400
	maxFileBytesForContext = 200 * 1024
	maxSelectedContexts    = 8
	maxCachedSectionChars  = 2200
	maxContextLineChars    = 180
)

func (c *Chat) buildCachedContextSection(projectPath, prompt string, availableList []string) string {
	if c.ContextStore == nil || projectPath == "" || len(availableList) == 0 {
		return ""
	}

	projectType := detectProjectProfile(availableList).Type
	intent := detectTaskIntent(prompt)
	candidates := pickRelevantFiles(availableList, prompt, projectType, intent, maxSelectedContexts)
	if len(candidates) == 0 {
		return ""
	}
	cacheByPath, _ := c.refreshFileContextCache(projectPath, candidates, nil)

	var b strings.Builder
	for _, p := range candidates {
		ctx := strings.TrimSpace(cacheByPath[p])
		if ctx == "" {
			continue
		}
		ctx = shorten(ctx, maxContextLineChars)
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString(": ")
		b.WriteString(ctx)
		b.WriteString("\n")
		if b.Len() >= maxCachedSectionChars {
			break
		}
	}
	return trimWithNotice(strings.TrimSpace(b.String()), maxCachedSectionChars, "\n...[truncated cached context]")
}

func (c *Chat) refreshFileContextCache(
	projectPath string,
	availableList []string,
	onProgress func(done, total int),
) (map[string]string, error) {
	cachedRecords, err := c.ContextStore.ListByProject(projectPath)
	if err != nil {
		return nil, err
	}

	cacheByPath := make(map[string]string, len(cachedRecords))
	metaByPath := make(map[string]struct {
		size int64
		mod  int64
	}, len(cachedRecords))
	for _, r := range cachedRecords {
		cacheByPath[r.FilePath] = r.Context
		metaByPath[r.FilePath] = struct {
			size int64
			mod  int64
		}{size: r.Size, mod: r.ModUnix}
	}

	limit := len(availableList)
	if limit > maxFilesToIndex {
		limit = maxFilesToIndex
	}
	total := limit
	done := 0

	for _, relPath := range availableList[:limit] {
		fullPath := filepath.Join(projectPath, relPath)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			done++
			if onProgress != nil {
				onProgress(done, total)
			}
			continue
		}

		size := info.Size()
		mod := info.ModTime().Unix()
		if prev, ok := metaByPath[relPath]; ok && prev.size == size && prev.mod == mod {
			done++
			if onProgress != nil {
				onProgress(done, total)
			}
			continue
		}

		context := summarizeFileContext(relPath, size, "")
		if size > 0 && size <= maxFileBytesForContext {
			content, err := c.Repo.ReadFile(projectPath, relPath)
			if err == nil {
				context = summarizeFileContext(relPath, size, content)
			}
		}

		_ = c.ContextStore.Upsert(projectPath, relPath, size, mod, context)
		cacheByPath[relPath] = context
		done++
		if onProgress != nil {
			onProgress(done, total)
		}
	}

	return cacheByPath, nil
}

func summarizeFileContext(path string, size int64, content string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if strings.TrimSpace(content) == "" {
		if size > maxFileBytesForContext {
			return fmt.Sprintf("%s file, %d KB. Skipped deep parsing due to size.", fileKind(ext), size/1024)
		}
		return fmt.Sprintf("%s file, %d bytes.", fileKind(ext), size)
	}

	lines := strings.Split(content, "\n")
	lineCount := len(lines)
	hints := extractHints(ext, lines, 6)
	if hints == "" {
		hints = "No major symbols detected."
	}
	return fmt.Sprintf("%s, %d lines. %s", fileKind(ext), lineCount, shorten(hints, 140))
}

func fileKind(ext string) string {
	switch ext {
	case ".go":
		return "Go source"
	case ".dart":
		return "Dart source"
	case ".py":
		return "Python source"
	case ".ts", ".tsx":
		return "TypeScript source"
	case ".js", ".jsx":
		return "JavaScript source"
	case ".md":
		return "Markdown doc"
	case ".json":
		return "JSON config"
	case ".yaml", ".yml":
		return "YAML config"
	default:
		return "Text/source"
	}
}

func extractHints(ext string, lines []string, maxHints int) string {
	hints := []string{}
	push := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, e := range hints {
			if e == v {
				return
			}
		}
		if len(hints) < maxHints {
			hints = append(hints, v)
		}
	}

	limit := len(lines)
	if limit > 300 {
		limit = 300
	}

	for _, raw := range lines[:limit] {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		switch ext {
		case ".go":
			if strings.HasPrefix(line, "package ") {
				push(line)
			}
			if strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "type ") {
				push(shorten(line, 100))
			}
		case ".py":
			if strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "class ") {
				push(shorten(line, 100))
			}
		case ".ts", ".tsx", ".js", ".jsx", ".dart":
			if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "function ") || strings.Contains(line, "=>") {
				push(shorten(line, 100))
			}
		case ".md":
			if strings.HasPrefix(line, "#") {
				push(shorten(line, 100))
			}
		default:
			if strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "def ") {
				push(shorten(line, 100))
			}
		}
		if len(hints) >= maxHints {
			break
		}
	}

	return strings.Join(hints, " | ")
}

func shorten(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
