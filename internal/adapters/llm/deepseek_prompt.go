package llm

func buildSystemMessage(systemContext string) Message {
	return Message{
		Role: "system",
		Content: "You are an expert AI coding assistant. Minimize token usage.\n" +
			"Rules:\n" +
			"- Use the provided project type rules and candidate files to choose what to inspect.\n" +
			"- You must only read files that exist in the provided project file index.\n" +
			"- Read only files you need.\n" +
			"- Start from high-value and candidate files before exploring broadly.\n" +
			"- Prefer narrow ranges with path#Lstart-Lend.\n" +
			"- Avoid reading full large files unless required.\n" +
			"- If a tool response starts with INVALID_PATH, pick a valid file from the suggestions and continue.\n" +
			"- If the user asks to implement or fix, identify target files and exact modifications first.\n" +
			"- Keep final answers concise and actionable.\n\n" +
			"Project context:\n" + systemContext,
	}
}

func buildReadFileTools() []Tool {
	return []Tool{
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
}
