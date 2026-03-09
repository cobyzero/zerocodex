package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *DeepSeekClient) Chat(systemContext, prompt string, readFunc func(string) string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY environment variable not set")
	}

	messages := []Message{
		buildSystemMessage(systemContext),
		{Role: "user", Content: prompt},
	}
	tools := buildReadFileTools()

	for i := 0; i < maxToolIterations; i++ {
		respMessage, err := c.doChatRequest(messages, tools)
		if err != nil {
			return "", err
		}

		messages = append(messages, respMessage)
		if len(respMessage.ToolCalls) == 0 {
			return respMessage.Content, nil
		}

		invalidPathCount := c.executeToolCalls(&messages, respMessage.ToolCalls, readFunc)
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
	return respMessage.Content, nil
}

func (c *DeepSeekClient) executeToolCalls(messages *[]Message, toolCalls []ToolCall, readFunc func(string) string) int {
	invalidPathCount := 0
	for _, tc := range toolCalls {
		if tc.Function.Name != "read_file" {
			continue
		}

		var args map[string]string
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			*messages = append(*messages, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Content:    "Error parsing arguments",
			})
			continue
		}

		content := readFunc(args["path"])
		if strings.HasPrefix(content, "INVALID_PATH:") {
			invalidPathCount++
		}

		*messages = append(*messages, Message{
			Role:       "tool",
			ToolCallID: tc.ID,
			Content:    content,
		})
	}
	return invalidPathCount
}
