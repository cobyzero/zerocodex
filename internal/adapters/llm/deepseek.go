package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DeepSeekClient struct {
	APIKey string
}

func NewDeepSeekClient() *DeepSeekClient {
	return &DeepSeekClient{
		APIKey: os.Getenv("DEEPSEEK_API_KEY"),
	}
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type ResponseBody struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func (c *DeepSeekClient) Chat(systemContext, prompt string, readFunc func(string) string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY environment variable not set")
	}

	url := "https://api.deepseek.com/chat/completions"

	messages := []Message{
		{
			Role: "system",
			Content: "You are an expert AI coding assistant. Minimize token usage.\n" +
				"Rules:\n" +
				"- Read only files you need.\n" +
				"- Prefer narrow ranges with path#Lstart-Lend.\n" +
				"- Avoid reading full large files unless required.\n" +
				"- Keep final answers concise and actionable.\n\n" +
				"Project context:\n" + systemContext,
		},
		{Role: "user", Content: prompt},
	}

	tools := []Tool{
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "read_file",
				Description: "Reads file content. You can request a line range by using path#Lstart-Lend (example: internal/main.go#L10-L80).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Relative file path. Optional line range format: path#Lstart-Lend.",
						},
					},
					"required": []string{"path"},
				},
			},
		},
	}

	for i := 0; i < 10; i++ { // Limit max tool call loops
		reqBody := RequestBody{
			Model:    "deepseek-chat", // DeepSeek-Chat represents the model supporting tool calls optimally
			Messages: messages,
			Tools:    tools,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return "", err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		bodyText, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("API error: %s", string(bodyText))
		}

		var respBody ResponseBody
		if err := json.Unmarshal(bodyText, &respBody); err != nil {
			return "", err
		}

		if len(respBody.Choices) == 0 {
			return "", fmt.Errorf("no response from DeepSeek API")
		}

		respMessage := respBody.Choices[0].Message
		messages = append(messages, respMessage)

		if len(respMessage.ToolCalls) == 0 {
			return respMessage.Content, nil
		}

		// Execute tools
		for _, tc := range respMessage.ToolCalls {
			if tc.Function.Name == "read_file" {
				var args map[string]string
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err == nil {
					content := readFunc(args["path"])
					messages = append(messages, Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    content,
					})
				} else {
					messages = append(messages, Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    "Error parsing arguments",
					})
				}
			}
		}
	}

	return "", fmt.Errorf("max tool execution iterations reached")
}
