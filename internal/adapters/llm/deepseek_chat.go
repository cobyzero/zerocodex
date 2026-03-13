package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *DeepSeekClient) Chat(
	systemContext,
	prompt string,
	readFunc func(string) string,
	writeFunc func(path, content string) string,
	runCommandFunc func(command string) string,
) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("deepseek api key is not configured")
	}

	messages := []Message{
		buildSystemMessage(systemContext),
		{Role: "user", Content: prompt},
	}
	tools := buildCodingTools()
	needsWrite := promptRequestsFileWrite(prompt)
	hasWritten := false
	forcedWriteAttempt := false

	for i := 0; i < maxToolIterations; i++ {
		respMessage, err := c.doChatRequest(messages, tools)
		if err != nil {
			return "", err
		}

		messages = append(messages, respMessage)
		if len(respMessage.ToolCalls) == 0 {
			if needsWrite && !hasWritten && !forcedWriteAttempt {
				forcedWriteAttempt = true
				messages = append(messages, Message{
					Role: "system",
					Content: "The user requested file modifications, but no write was applied yet. " +
						"You must call write_file now with the final full content of the target file(s).",
				})
				continue
			}
			return sanitizeAssistantContent(respMessage.Content), nil
		}

		invalidPathCount, writeOKCount := c.executeToolCalls(&messages, respMessage.ToolCalls, readFunc, writeFunc, runCommandFunc)
		if writeOKCount > 0 {
			hasWritten = true
		}
		if invalidPathCount > 0 {
			messages = append(messages, Message{
				Role: "system",
				Content: "Some requested paths were invalid. Use only existing paths from the project index and tool suggestions. " +
					"If enough information is available, provide the final answer now without more file reads.",
			})
		}
	}

	messages = append(messages, Message{
		Role: "system",
		Content: "Stop calling tools. Provide the best possible final answer now using available information " +
			"and clearly mark assumptions when needed.",
	})

	respMessage, err := c.doChatRequest(messages, nil)
	if err != nil {
		return "", err
	}
	return sanitizeAssistantContent(respMessage.Content), nil
}

func (c *DeepSeekClient) executeToolCalls(
	messages *[]Message,
	toolCalls []ToolCall,
	readFunc func(string) string,
	writeFunc func(path, content string) string,
	runCommandFunc func(command string) string,
) (invalidPathCount, writeOKCount int) {
	invalidPathCount = 0
	writeOKCount = 0
	for _, tc := range toolCalls {
		var toolContent string

		switch tc.Function.Name {
		case "read_file":
			var args map[string]string
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				toolContent = "Error parsing arguments"
				break
			}
			toolContent = readFunc(args["path"])
			if strings.HasPrefix(toolContent, "INVALID_PATH:") {
				invalidPathCount++
			}

		case "write_file":
			var args struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				toolContent = "Error parsing arguments"
				break
			}
			toolContent = writeFunc(args.Path, args.Content)
			if strings.HasPrefix(toolContent, "INVALID_PATH:") {
				invalidPathCount++
			}
			if strings.HasPrefix(toolContent, "WRITE_OK:") {
				writeOKCount++
			}

		case "run_command":
			var args struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				toolContent = "Error parsing arguments"
				break
			}
			toolContent = runCommandFunc(args.Command)

		default:
			toolContent = "Unsupported tool: " + tc.Function.Name
		}

		*messages = append(*messages, Message{
			Role:       "tool",
			ToolCallID: tc.ID,
			Content:    toolContent,
		})
	}
	return invalidPathCount, writeOKCount
}

func promptRequestsFileWrite(prompt string) bool {
	p := strings.ToLower(prompt)
	keywords := []string{
		"modifica", "modificar", "actualiza", "actualizar", "edita", "editar",
		"reescribe", "escribe", "documenta", "documentar", "crea", "crear",
		"fix", "update", "edit", "modify", "rewrite", "write", "implement",
	}
	for _, kw := range keywords {
		if strings.Contains(p, kw) {
			return true
		}
	}
	return false
}
