package application

import (
	"fmt"
	"regexp"
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
	if projectPath != "" {
		files := c.Repo.ListFiles(projectPath)
		context = buildAgentContext(files, prompt)
		context = trimWithNotice(context, maxContextChars, "\n...[truncated context]")
	}

	return c.Client.Chat(context, prompt, func(file string) string {
		if onTool != nil {
			onTool("Reading file: " + file)
		}
		path, startLine, endLine := parseFileRange(file)
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
