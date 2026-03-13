package llm

import "strings"

func sanitizeAssistantContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	skippingBlock := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)

		if strings.Contains(line, "DSML") || strings.Contains(line, "invoke name=") || strings.Contains(line, "function_calls") {
			skippingBlock = true
			continue
		}
		if skippingBlock {
			if line == "" || strings.Contains(line, "</") || strings.Contains(line, "invoke>") {
				continue
			}
			// Stop skipping once normal prose resumes.
			if !strings.Contains(line, "parameter name=") && !strings.Contains(line, "string=") {
				skippingBlock = false
			} else {
				continue
			}
		}

		if strings.HasPrefix(line, "<|") || strings.HasPrefix(line, "<｜") || strings.HasPrefix(line, "</") {
			continue
		}
		if line == "/" || line == "[!NOTE]" || line == "]" {
			continue
		}

		kept = append(kept, raw)
	}

	return strings.TrimSpace(strings.Join(kept, "\n"))
}
