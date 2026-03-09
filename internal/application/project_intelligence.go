package application

import (
	"sort"
	"strings"
	"unicode"
)

type projectProfile struct {
	Type     string
	Rules    []string
	KeyFiles []string
}

func buildAgentContext(filesListRaw, prompt string) string {
	paths := parseListedFiles(filesListRaw)
	profile := detectProjectProfile(paths)
	candidates := pickRelevantFiles(paths, prompt, profile.Type, 18)

	var b strings.Builder
	b.WriteString("Project Type: ")
	b.WriteString(profile.Type)
	b.WriteString("\n\n")

	b.WriteString("Rules for this project type:\n")
	for _, rule := range profile.Rules {
		b.WriteString("- ")
		b.WriteString(rule)
		b.WriteString("\n")
	}

	b.WriteString("\nHigh-value project files:\n")
	for _, p := range profile.KeyFiles {
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString("\n")
	}

	if len(candidates) > 0 {
		b.WriteString("\nBest candidate files for this request:\n")
		for _, p := range candidates {
			b.WriteString("- ")
			b.WriteString(p)
			b.WriteString("\n")
		}
	}

	b.WriteString("\nProject files (full listing, possibly truncated):\n")
	b.WriteString(filesListRaw)
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
			},
			KeyFiles: existingFrom(paths, "go.mod", "cmd/app/main.go", "README.md"),
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
			},
			KeyFiles: existingFrom(paths, "package.json", "tsconfig.json", "README.md"),
		}
	case has("pyproject.toml") || has("requirements.txt"):
		return projectProfile{
			Type: "Python",
			Rules: []string{
				"Identify package layout first (src/, app/, or module root) before editing.",
				"Preserve virtual-env and dependency conventions in pyproject.toml/requirements.txt.",
				"Keep business logic separated from transport layer (CLI/API/web).",
				"Prefer focused test updates for changed behavior.",
			},
			KeyFiles: existingFrom(paths, "pyproject.toml", "requirements.txt", "README.md"),
		}
	case has("cargo.toml"):
		return projectProfile{
			Type: "Rust",
			Rules: []string{
				"Respect crate/module boundaries and ownership-driven API design.",
				"Prefer compile-safe refactors and explicit error handling via Result.",
				"Keep changes minimal in public API unless requested.",
				"Update tests in mod tests or integration tests when behavior changes.",
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
			KeyFiles: existingFrom(paths, "README.md"),
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

func pickRelevantFiles(paths []string, prompt, projectType string, limit int) []string {
	type scored struct {
		path  string
		score int
	}

	prompt = strings.ToLower(prompt)
	tokens := tokenize(prompt)
	scoredFiles := make([]scored, 0, len(paths))

	for _, p := range paths {
		lp := strings.ToLower(p)
		score := scoreByProjectType(lp, projectType) + scoreByPromptHints(lp, prompt)
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

func scoreByProjectType(path, projectType string) int {
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
		case strings.Contains(path, "readme"):
			return 2
		}
	case "Flutter/Dart":
		switch {
		case path == "pubspec.yaml":
			return 8
		case strings.HasPrefix(path, "lib/"):
			return 6
		case strings.HasSuffix(path, ".dart"):
			return 5
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
		}
	case "Python":
		switch {
		case path == "pyproject.toml" || path == "requirements.txt":
			return 8
		case strings.HasSuffix(path, ".py"):
			return 5
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
		}
	}

	if strings.Contains(path, "readme") {
		return 2
	}
	return 0
}

func scoreByPromptHints(path, prompt string) int {
	score := 0
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
	return score
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
