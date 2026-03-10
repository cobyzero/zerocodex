package ui

import (
	"bytes"
	"os/exec"
	"strconv"
	"strings"
)

type fileChange struct {
	path    string
	added   int
	deleted int
}

func buildGitChangesMarkdown(projectPath string) string {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return ""
	}

	if !isGitRepo(projectPath) {
		return ""
	}

	changes := collectNumStat(projectPath, "diff", "--numstat")
	staged := collectNumStat(projectPath, "diff", "--cached", "--numstat")
	changes = mergeChanges(changes, staged)
	if len(changes) == 0 {
		return ""
	}

	totalAdded := 0
	totalDeleted := 0
	for _, c := range changes {
		totalAdded += c.added
		totalDeleted += c.deleted
	}

	var b strings.Builder
	b.WriteString("## 📊 Cambios en Git\n\n")
	
	// Resumen con emojis y formato mejorado
	b.WriteString("**📁 ")
	b.WriteString(strconv.Itoa(len(changes)))
	b.WriteString(" archivos modificados**\n\n")
	
	// Estadísticas con colores
	b.WriteString("```diff\n")
	b.WriteString("+")
	b.WriteString(strconv.Itoa(totalAdded))
	b.WriteString(" líneas añadidas  |  -")
	b.WriteString(strconv.Itoa(totalDeleted))
	b.WriteString(" líneas eliminadas\n")
	b.WriteString("```\n\n")
	
	// Lista de archivos con formato mejorado
	b.WriteString("### Archivos modificados:\n\n")
	for _, c := range changes {
		b.WriteString("• `")
		b.WriteString(c.path)
		b.WriteString("` ")
		
		// Iconos según el tipo de cambios
		if c.added > 0 && c.deleted > 0 {
			b.WriteString("🔄 ")
		} else if c.added > 0 {
			b.WriteString("➕ ")
		} else if c.deleted > 0 {
			b.WriteString("➖ ")
		}
		
		b.WriteString("`+")
		b.WriteString(strconv.Itoa(c.added))
		b.WriteString(" -")
		b.WriteString(strconv.Itoa(c.deleted))
		b.WriteString("`\n")
	}
	
	// Footer con información adicional
	b.WriteString("\n---\n")
	b.WriteString("*Los cambios mostrados incluyen tanto los staged como los unstaged.*")

	return b.String()
}

func isGitRepo(projectPath string) bool {
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--is-inside-work-tree")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) == "true"
}

func collectNumStat(projectPath string, args ...string) []fileChange {
	fullArgs := append([]string{"-C", projectPath}, args...)
	cmd := exec.Command("git", fullArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	changes := make([]fileChange, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		added, errA := strconv.Atoi(parts[0])
		deleted, errD := strconv.Atoi(parts[1])
		if errA != nil || errD != nil {
			continue
		}

		changes = append(changes, fileChange{
			path:    parts[2],
			added:   added,
			deleted: deleted,
		})
	}
	return changes
}

func mergeChanges(base, extra []fileChange) []fileChange {
	idx := make(map[string]int, len(base))
	out := make([]fileChange, 0, len(base)+len(extra))
	for _, c := range base {
		idx[c.path] = len(out)
		out = append(out, c)
	}

	for _, c := range extra {
		if i, ok := idx[c.path]; ok {
			out[i].added += c.added
			out[i].deleted += c.deleted
			continue
		}
		idx[c.path] = len(out)
		out = append(out, c)
	}
	return out
}
