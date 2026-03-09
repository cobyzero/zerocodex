package application

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cobyzero/zerocodex/internal/domain"
)

const (
	maxContextChars  = 12000
	maxToolReadChars = 20000
)

var lineRangePattern = regexp.MustCompile(`^(.*)#L(\d+)-L(\d+)$`)

type Chat struct {
	Repo   domain.ProjectRepository
	Client domain.LLMClient
}

func (c *Chat) Execute(projectPath, prompt string, onTool func(string)) (string, error) {
	var context string
	availableFiles := map[string]string{}
	availableList := []string{}
	if projectPath != "" {
		files := c.Repo.ListFiles(projectPath)
		context = buildAgentContext(files, prompt)
		context = trimWithNotice(context, maxContextChars, "\n...[truncated context]")

		availableList = parseListedFiles(files)
		availableFiles = make(map[string]string, len(availableList))
		for _, p := range availableList {
			availableFiles[strings.ToLower(strings.TrimSpace(p))] = p
		}
	}

	return c.Client.Chat(context, prompt, func(file string) string {
		if onTool != nil {
			onTool("Reading file: " + file)
		}
		path, startLine, endLine := parseFileRange(file)
		normalized := strings.ToLower(strings.TrimSpace(path))
		if len(availableFiles) > 0 {
			if _, ok := availableFiles[normalized]; !ok {
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
				return b.String()
			}
			path = availableFiles[normalized]
		}

		content, err := c.Repo.ReadFile(projectPath, path)
		if err != nil {
			return "Error reading file: " + err.Error()
		}
		if startLine > 0 && endLine >= startLine {
			content = sliceLines(content, startLine, endLine)
		}
		content = trimWithNotice(content, maxToolReadChars, "\n...[truncated file content]")
		return content
	})
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
