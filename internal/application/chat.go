package application

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cobyzero/zerocodex/internal/domain"
)

const (
	maxContextChars    = 8500
	maxToolReadChars   = 12000
	maxToolExecChars   = 12000
	maxHistoryChars    = 1600
	maxHistoryItems    = 5
	maxHistoryEntry    = 140
	toolCommandTimeout = 4 * time.Minute
)

var lineRangePattern = regexp.MustCompile(`^(.*)#L(\d+)-L(\d+)$`)

type Chat struct {
	Repo         domain.ProjectRepository
	Client       domain.LLMClient
	ContextStore domain.FileContextStore
	HistoryStore domain.ChatHistoryStore
}

type projectRangeReader interface {
	ReadFileRange(basePath, relPath string, startLine, endLine int) (string, error)
}

func (c *Chat) AnalyzeProject(projectPath string, onProgress func(done, total int)) error {
	if strings.TrimSpace(projectPath) == "" || c.ContextStore == nil {
		return nil
	}
	files := c.Repo.ListFiles(projectPath)
	availableList := parseListedFiles(files)
	_, err := c.refreshFileContextCache(projectPath, availableList, onProgress)
	return err
}

func (c *Chat) Execute(projectPath, prompt string, onTool func(string)) (string, error) {
	var context string
	availableFiles := map[string]string{}
	availableList := []string{}
	readCache := map[string]string{}
	if projectPath != "" {
		files := c.Repo.ListFiles(projectPath)
		availableList = parseListedFiles(files)
		availableFiles = make(map[string]string, len(availableList))
		for _, p := range availableList {
			availableFiles[strings.ToLower(strings.TrimSpace(p))] = p
		}

		context = buildAgentContext(files, prompt)
		if cacheSection := c.buildCachedContextSection(projectPath, prompt, availableList); cacheSection != "" {
			context = context + "\n\nCached file context (local, auto-maintained):\n" + cacheSection
		}
		if historySection := c.buildHistoryContextSection(projectPath); historySection != "" {
			context = context + "\n\nRecent chat history for this project:\n" + historySection
		}
		context = trimWithNotice(context, maxContextChars, "\n...[truncated context]")
	}

	if c.HistoryStore != nil && strings.TrimSpace(projectPath) != "" && strings.TrimSpace(prompt) != "" {
		_ = c.HistoryStore.Append(projectPath, "user", prompt)
	}

	response, err := c.Client.Chat(
		context,
		prompt,
		func(file string) string {
			if onTool != nil {
				onTool("Reading file: " + file)
			}
			if _, ok := readCache[file]; ok {
				return "ALREADY_PROVIDED: " + file + "\nUse the previous tool result for this file/range."
			}
			path, startLine, endLine := parseFileRange(file)
			resolvedPath, ok, invalidMessage := resolveExistingPath(path, availableFiles, availableList)
			if !ok {
				return invalidMessage
			}

			var content string
			var err error
			if startLine > 0 && endLine >= startLine {
				if rr, ok := c.Repo.(projectRangeReader); ok {
					content, err = rr.ReadFileRange(projectPath, resolvedPath, startLine, endLine)
				} else {
					content, err = c.Repo.ReadFile(projectPath, resolvedPath)
					if err == nil {
						content = sliceLines(content, startLine, endLine)
					}
				}
			} else {
				content, err = c.Repo.ReadFile(projectPath, resolvedPath)
			}
			if err != nil {
				return "Error reading file: " + err.Error()
			}
			content = trimWithNotice(content, maxToolReadChars, "\n...[truncated file content]")
			readCache[file] = content
			return content
		},
		func(path, content string) string {
			if onTool != nil {
				onTool("Writing file: " + path)
			}
			resolvedPath, ok, invalidMessage := resolveWritablePath(path, projectPath, availableFiles)
			if !ok {
				return invalidMessage
			}
			if err := c.Repo.WriteFile(projectPath, resolvedPath, content); err != nil {
				return "WRITE_ERROR: " + err.Error()
			}
			return "WRITE_OK: " + resolvedPath
		},
		func(command string) string {
			if onTool != nil {
				onTool("Running command: " + strings.TrimSpace(command))
			}
			return executeProjectCommand(projectPath, command)
		},
	)
	if err != nil {
		if c.HistoryStore != nil && strings.TrimSpace(projectPath) != "" {
			_ = c.HistoryStore.Append(projectPath, "error", err.Error())
		}
		return "", err
	}
	if c.HistoryStore != nil && strings.TrimSpace(projectPath) != "" && strings.TrimSpace(response) != "" {
		_ = c.HistoryStore.Append(projectPath, "assistant", response)
	}
	return response, nil
}

func executeProjectCommand(projectPath, command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return "COMMAND_ERROR: empty command"
	}
	if strings.TrimSpace(projectPath) == "" {
		return "COMMAND_ERROR: no project selected"
	}

	ctx, cancel := context.WithTimeout(context.Background(), toolCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	cmd.Dir = projectPath

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	errOut := strings.TrimSpace(stderr.String())

	var b strings.Builder
	if err == nil {
		b.WriteString("COMMAND_OK\n")
	} else if ctx.Err() == context.DeadlineExceeded {
		b.WriteString("COMMAND_TIMEOUT: exceeded ")
		b.WriteString(toolCommandTimeout.String())
		b.WriteString("\n")
	} else {
		b.WriteString("COMMAND_ERROR: ")
		b.WriteString(err.Error())
		b.WriteString("\n")
	}

	if out != "" {
		b.WriteString("stdout:\n")
		b.WriteString(out)
		b.WriteString("\n")
	}
	if errOut != "" {
		b.WriteString("stderr:\n")
		b.WriteString(errOut)
		b.WriteString("\n")
	}

	return trimWithNotice(strings.TrimSpace(b.String()), maxToolExecChars, "\n...[truncated command output]")
}

func parseFileRange(raw string) (path string, startLine, endLine int) {
	matches := lineRangePattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 4 {
		return raw, 0, 0
	}

	start, errStart := strconv.Atoi(matches[2])
	end, errEnd := strconv.Atoi(matches[3])
	if errStart != nil || errEnd != nil || start < 1 || end < start {
		return raw, 0, 0
	}
	return strings.TrimSpace(matches[1]), start, end
}

func sliceLines(content string, startLine, endLine int) string {
	lines := strings.Split(content, "\n")
	if startLine > len(lines) {
		return fmt.Sprintf("Requested range L%d-L%d is out of bounds. File has %d lines.", startLine, endLine, len(lines))
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	return strings.Join(lines[startLine-1:endLine], "\n")
}

func trimWithNotice(s string, maxChars int, notice string) string {
	if len(s) <= maxChars {
		return s
	}
	cut := maxChars - len(notice)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + notice
}

func suggestFilePaths(requested string, available []string, limit int) []string {
	requested = strings.ToLower(strings.TrimSpace(requested))
	base := strings.ToLower(filepath.Base(requested))

	type cand struct {
		path  string
		score int
	}
	scored := make([]cand, 0, len(available))

	for _, p := range available {
		lp := strings.ToLower(p)
		score := 0
		if strings.Contains(lp, requested) || strings.Contains(requested, lp) {
			score += 6
		}
		if base != "" {
			pbase := strings.ToLower(filepath.Base(lp))
			if pbase == base {
				score += 8
			} else if strings.Contains(pbase, base) || strings.Contains(base, pbase) {
				score += 4
			}
		}
		if strings.HasSuffix(lp, ".md") && strings.HasSuffix(requested, ".md") {
			score += 1
		}
		if score > 0 {
			scored = append(scored, cand{path: p, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].path < scored[j].path
		}
		return scored[i].score > scored[j].score
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	out := make([]string, 0, len(scored))
	for _, c := range scored {
		out = append(out, c.path)
	}
	return out
}

func resolveExistingPath(path string, availableFiles map[string]string, availableList []string) (string, bool, string) {
	normalized := strings.ToLower(strings.TrimSpace(path))
	if len(availableFiles) > 0 {
		if p, ok := availableFiles[normalized]; ok {
			return p, true, ""
		}
	}

	suggestions := suggestFilePaths(path, availableList, 8)
	var b strings.Builder
	b.WriteString("INVALID_PATH: ")
	b.WriteString(path)
	b.WriteString("\nThe requested file is not in this project index.")
	if len(suggestions) > 0 {
		b.WriteString("\nTry one of these existing paths:\n")
		for _, s := range suggestions {
			b.WriteString("- ")
			b.WriteString(s)
			b.WriteString("\n")
		}
	}
	return "", false, b.String()
}

func resolveWritablePath(path, basePath string, availableFiles map[string]string) (string, bool, string) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", false, "INVALID_PATH: empty path"
	}

	if p, ok := availableFiles[strings.ToLower(trimmed)]; ok {
		return p, true, ""
	}

	clean := filepath.Clean(trimmed)
	if filepath.IsAbs(clean) {
		return "", false, "INVALID_PATH: absolute paths are not allowed"
	}
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
		return "", false, "INVALID_PATH: path traversal is not allowed"
	}

	full := filepath.Join(basePath, clean)
	absBase, errBase := filepath.Abs(basePath)
	absFull, errFull := filepath.Abs(full)
	if errBase != nil || errFull != nil {
		return "", false, "INVALID_PATH: unable to validate path"
	}
	if absFull != absBase && !strings.HasPrefix(absFull, absBase+string(os.PathSeparator)) {
		return "", false, "INVALID_PATH: path is outside project root"
	}

	// Allow creating new files inside the selected project.
	return clean, true, ""
}

func (c *Chat) LoadProjectTranscript(projectPath string) string {
	if c.HistoryStore == nil || strings.TrimSpace(projectPath) == "" {
		return ""
	}
	entries, err := c.HistoryStore.ListRecent(projectPath, 100)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	for i, entry := range entries {
		if i > 0 {
			b.WriteString("\n\n")
		}
		switch entry.Role {
		case "user":
			b.WriteString("#### You\n\n> ")
			b.WriteString(strings.TrimSpace(entry.Content))
		case "assistant":
			b.WriteString("#### ZeroCodex\n\n")
			b.WriteString(strings.TrimSpace(entry.Content))
		case "error":
			b.WriteString("`Error` ")
			b.WriteString(strings.TrimSpace(entry.Content))
		}
	}
	return strings.TrimSpace(b.String())
}

func (c *Chat) buildHistoryContextSection(projectPath string) string {
	if c.HistoryStore == nil || strings.TrimSpace(projectPath) == "" {
		return ""
	}
	entries, err := c.HistoryStore.ListRecent(projectPath, maxHistoryItems)
	if err != nil || len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	for _, entry := range entries {
		role := entry.Role
		if entry.Role == "assistant" {
			role = "Assistant"
		} else if entry.Role == "user" {
			role = "User"
		} else if entry.Role == "error" {
			role = "Error"
		}
		b.WriteString("- ")
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(trimWithNotice(strings.TrimSpace(entry.Content), maxHistoryEntry, "..."))
		b.WriteString("\n")
	}
	return trimWithNotice(strings.TrimSpace(b.String()), maxHistoryChars, "\n...[truncated history]")
}

func (c *Chat) ClearProjectHistory(projectPath string) error {
	if c.HistoryStore == nil || strings.TrimSpace(projectPath) == "" {
		return nil
	}
	return c.HistoryStore.Clear(projectPath)
}
