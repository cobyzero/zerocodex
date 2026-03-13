package application

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type projectProfile struct {
	Type     string
	Rules    []string
	KeyFiles []string
}

const (
	maxContextRules      = 3
	maxContextKeyFiles   = 4
	maxContextCandidates = 8
	maxInventoryRoot     = 6
	maxInventoryDirs     = 8
)

func buildAgentContext(filesListRaw, prompt string) string {
	paths := parseListedFiles(filesListRaw)
	intent := detectTaskIntent(prompt)
	profile := detectProjectProfile(paths)
	candidates := pickRelevantFiles(paths, prompt, profile.Type, intent, maxContextCandidates)

	var b strings.Builder
	b.WriteString("Project Type: ")
	b.WriteString(profile.Type)
	b.WriteByte('\n')

	b.WriteString("Rules:\n")
	for _, rule := range capStrings(profile.Rules, maxContextRules) {
		b.WriteString("- ")
		b.WriteString(rule)
		b.WriteString("\n")
	}

	if len(profile.KeyFiles) > 0 {
		b.WriteString("\nKey files:\n- ")
		b.WriteString(strings.Join(capStrings(profile.KeyFiles, maxContextKeyFiles), ", "))
		b.WriteString("\n")
	}

	if len(candidates) > 0 {
		b.WriteString("\nCandidate files:\n- ")
		b.WriteString(strings.Join(candidates, ", "))
		b.WriteString("\n")
	}

	if inventory := buildCompactInventory(paths); inventory != "" {
		b.WriteString("\nInventory:\n")
		b.WriteString(inventory)
	}
	return b.String()
}

func parseListedFiles(raw string) []string {
	lines := strings.Split(raw, "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			paths = append(paths, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		}
	}
	return paths
}

func detectProjectProfile(paths []string) projectProfile {
	pathSet := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		pathSet[strings.ToLower(p)] = struct{}{}
	}

	has := func(path string) bool {
		_, ok := pathSet[strings.ToLower(path)]
		return ok
	}

	switch {
	case has("pubspec.yaml"):
		return projectProfile{
			Type: "Flutter/Dart",
			Rules: []string{
				"Prioritize files under lib/, especially feature and view/widget layers.",
				"Keep UI logic in widgets and business logic in services/controllers/providers.",
				"Respect pubspec.yaml dependencies and existing state-management pattern.",
				"For UI work, check theme/shared widgets before duplicating styles.",
			},
			KeyFiles: existingFrom(paths, "pubspec.yaml", "lib/main.dart", "analysis_options.yaml"),
		}
	case has("go.mod"):
		return projectProfile{
			Type: "Go",
			Rules: []string{
				"Follow package boundaries: cmd/ for entrypoints, internal/ for app logic, adapters for IO.",
				"Keep interfaces in domain/application layers and concrete code in adapters.",
				"Prefer small focused functions; keep error handling explicit.",
				"When changing behavior, update relevant tests or add missing ones.",
				"Do not start with README unless the task is documentation or general project explanation.",
			},
			KeyFiles: existingFrom(paths, "go.mod", "cmd/app/main.go"),
		}
	case has("package.json"):
		tp := "Node.js"
		if has("next.config.js") || has("next.config.mjs") || has("next.config.ts") {
			tp = "Next.js"
		}
		return projectProfile{
			Type: tp,
			Rules: []string{
				"Check package.json scripts and dependencies before making structural changes.",
				"Respect current framework conventions (routing, components, state, build tools).",
				"Prefer updating files inside src/ unless project already uses root-level app structure.",
				"Keep lint/format conventions and existing naming patterns.",
				"Do not start with README unless the task is documentation or general project explanation.",
			},
			KeyFiles: existingFrom(paths, "package.json", "tsconfig.json"),
		}
	case has("pyproject.toml") || has("requirements.txt"):
		return projectProfile{
			Type: "Python",
			Rules: []string{
				"Identify package layout first (src/, app/, or module root) before editing.",
				"Preserve virtual-env and dependency conventions in pyproject.toml/requirements.txt.",
				"Keep business logic separated from transport layer (CLI/API/web).",
				"Prefer focused test updates for changed behavior.",
				"Do not start with README unless the task is documentation or general project explanation.",
			},
			KeyFiles: existingFrom(paths, "pyproject.toml", "requirements.txt"),
		}
	case has("cargo.toml"):
		return projectProfile{
			Type: "Rust",
			Rules: []string{
				"Respect crate/module boundaries and ownership-driven API design.",
				"Prefer compile-safe refactors and explicit error handling via Result.",
				"Keep changes minimal in public API unless requested.",
				"Update tests in mod tests or integration tests when behavior changes.",
				"Do not start with README unless the task is documentation or general project explanation.",
			},
			KeyFiles: existingFrom(paths, "Cargo.toml", "src/main.rs", "src/lib.rs"),
		}
	default:
		return projectProfile{
			Type: "Generic",
			Rules: []string{
				"Start with configuration and entrypoint files to map architecture.",
				"Read the minimum files needed to complete the task.",
				"Prefer coherent, localized changes over broad refactors.",
				"State assumptions when project conventions are unclear.",
			},
			KeyFiles: existingFrom(paths),
		}
	}
}

func existingFrom(paths []string, candidates ...string) []string {
	set := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		set[strings.ToLower(p)] = struct{}{}
	}

	found := make([]string, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := set[strings.ToLower(c)]; ok {
			found = append(found, c)
		}
	}
	return found
}

func pickRelevantFiles(paths []string, prompt, projectType string, intent taskIntent, limit int) []string {
	type scored struct {
		path  string
		score int
	}

	prompt = strings.ToLower(prompt)
	tokens := tokenize(prompt)
	scoredFiles := make([]scored, 0, len(paths))

	for _, p := range paths {
		lp := strings.ToLower(p)
		score := scoreByProjectType(lp, projectType, intent) + scoreByPromptHints(lp, prompt, intent)
		for _, token := range tokens {
			if len(token) < 3 {
				continue
			}
			if strings.Contains(lp, token) {
				score += 2
			}
		}
		if score > 0 {
			scoredFiles = append(scoredFiles, scored{path: p, score: score})
		}
	}

	sort.Slice(scoredFiles, func(i, j int) bool {
		if scoredFiles[i].score == scoredFiles[j].score {
			return scoredFiles[i].path < scoredFiles[j].path
		}
		return scoredFiles[i].score > scoredFiles[j].score
	})

	if len(scoredFiles) > limit {
		scoredFiles = scoredFiles[:limit]
	}

	out := make([]string, 0, len(scoredFiles))
	for _, item := range scoredFiles {
		out = append(out, item.path)
	}
	return out
}

func buildCompactInventory(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	topDirCounts := map[string]int{}
	rootFiles := []string{}
	for _, path := range paths {
		if path == "" {
			continue
		}
		parts := strings.Split(path, "/")
		if len(parts) == 1 {
			rootFiles = append(rootFiles, path)
			continue
		}
		topDirCounts[parts[0]]++
	}

	rootFiles = capStrings(rootFiles, maxInventoryRoot)
	type dirCount struct {
		name  string
		count int
	}
	dirs := make([]dirCount, 0, len(topDirCounts))
	for name, count := range topDirCounts {
		dirs = append(dirs, dirCount{name: name, count: count})
	}
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i].count == dirs[j].count {
			return dirs[i].name < dirs[j].name
		}
		return dirs[i].count > dirs[j].count
	})
	if len(dirs) > maxInventoryDirs {
		dirs = dirs[:maxInventoryDirs]
	}

	var b strings.Builder
	if len(rootFiles) > 0 {
		b.WriteString("- root files: ")
		b.WriteString(strings.Join(rootFiles, ", "))
		b.WriteString("\n")
	}
	for _, item := range dirs {
		b.WriteString("- ")
		b.WriteString(item.name)
		b.WriteString("/: ")
		b.WriteString(strconv.Itoa(item.count))
		b.WriteString(" files\n")
	}
	return b.String()
}

func scoreByProjectType(path, projectType string, intent taskIntent) int {
	switch projectType {
	case "Go":
		switch {
		case path == "go.mod":
			return 8
		case strings.HasPrefix(path, "cmd/"):
			return 5
		case strings.HasPrefix(path, "internal/"):
			return 5
		case strings.HasSuffix(path, ".go"):
			return 4
		case isReadmePath(path):
			return readmeScore(intent)
		}
	case "Flutter/Dart":
		switch {
		case path == "pubspec.yaml":
			return 8
		case strings.HasPrefix(path, "lib/"):
			return 6
		case strings.HasSuffix(path, ".dart"):
			return 5
		case isReadmePath(path):
			return readmeScore(intent)
		case strings.Contains(path, "test"):
			return 3
		}
	case "Next.js", "Node.js":
		switch {
		case path == "package.json":
			return 8
		case strings.HasPrefix(path, "src/"):
			return 6
		case strings.Contains(path, "app/") || strings.Contains(path, "pages/"):
			return 5
		case strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".jsx"):
			return 4
		case isReadmePath(path):
			return readmeScore(intent)
		}
	case "Python":
		switch {
		case path == "pyproject.toml" || path == "requirements.txt":
			return 8
		case strings.HasSuffix(path, ".py"):
			return 5
		case isReadmePath(path):
			return readmeScore(intent)
		case strings.Contains(path, "tests"):
			return 3
		}
	case "Rust":
		switch {
		case path == "cargo.toml":
			return 8
		case strings.HasPrefix(path, "src/"):
			return 6
		case strings.HasSuffix(path, ".rs"):
			return 5
		case isReadmePath(path):
			return readmeScore(intent)
		}
	}

	if isReadmePath(path) {
		return readmeScore(intent)
	}
	return 0
}

func scoreByPromptHints(path, prompt string, intent taskIntent) int {
	score := 0
	if intent == intentDocumentation && isReadmePath(path) {
		score += 8
	}
	if strings.Contains(prompt, "test") && (strings.Contains(path, "test") || strings.Contains(path, "_test")) {
		score += 4
	}
	if strings.Contains(prompt, "ui") || strings.Contains(prompt, "interfaz") || strings.Contains(prompt, "estilo") {
		if strings.Contains(path, "ui") || strings.Contains(path, "view") || strings.Contains(path, "widget") || strings.Contains(path, "theme") {
			score += 4
		}
	}
	if strings.Contains(prompt, "api") || strings.Contains(prompt, "endpoint") || strings.Contains(prompt, "http") {
		if strings.Contains(path, "api") || strings.Contains(path, "handler") || strings.Contains(path, "controller") || strings.Contains(path, "route") {
			score += 4
		}
	}
	if strings.Contains(prompt, "config") || strings.Contains(prompt, "setup") {
		if strings.Contains(path, "config") || strings.Contains(path, ".env") || strings.Contains(path, "settings") {
			score += 3
		}
	}
	if intent == intentImplementation || intent == intentBugfix {
		if isReadmePath(path) {
			score -= 6
		}
	}
	return score
}

type taskIntent string

const (
	intentGeneric        taskIntent = "generic"
	intentDocumentation  taskIntent = "documentation"
	intentImplementation taskIntent = "implementation"
	intentBugfix         taskIntent = "bugfix"
	intentUI             taskIntent = "ui"
)

func detectTaskIntent(prompt string) taskIntent {
	prompt = strings.ToLower(prompt)
	switch {
	case hasAny(prompt, "readme", "documenta", "documentar", "document", "documentation", "docs", "explica el proyecto", "como funciona el proyecto"):
		return intentDocumentation
	case hasAny(prompt, "fix", "bug", "error", "falla", "arregla", "corrige"):
		return intentBugfix
	case hasAny(prompt, "ui", "interfaz", "estilo", "diseño", "design"):
		return intentUI
	case hasAny(prompt, "implement", "modifica", "actualiza", "edita", "crear", "crea", "rewrite", "write", "refactor"):
		return intentImplementation
	default:
		return intentGeneric
	}
}

func hasAny(s string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

func isReadmePath(path string) bool {
	return strings.Contains(path, "readme")
}

func readmeScore(intent taskIntent) int {
	switch intent {
	case intentDocumentation:
		return 8
	case intentGeneric:
		return 1
	default:
		return -4
	}
}

func tokenize(s string) []string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Fields(b.String())
}

func capStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
