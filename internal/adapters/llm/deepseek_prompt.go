package llm

func buildSystemMessage(systemContext string) Message {
	return Message{
		Role: "system",
		Content: "You are an expert AI coding assistant. Minimize token usage.\n" +
			"Rules:\n" +
			"- Use the provided project type rules and candidate files to choose what to inspect.\n" +
			"- Do not read README first unless the task is documentation, onboarding, or a general explanation request.\n" +
			"- You must only read files that exist in the provided project file index.\n" +
			"- Read only files you need.\n" +
			"- Prefer using cached file context and candidate lists before requesting new file reads.\n" +
			"- Start from high-value and candidate files before exploring broadly.\n" +
			"- Avoid rereading the same file or range if it was already provided in this turn.\n" +
			"- Prefer narrow ranges with path#Lstart-Lend.\n" +
			"- When the user asks to modify files, call write_file with the full updated content.\n" +
			"- Use run_command when a task requires terminal actions (install, build, scaffold, generate).\n" +
			"- Avoid reading full large files unless required.\n" +
			"- If a tool response starts with INVALID_PATH, pick a valid file from the suggestions and continue.\n" +
			"- If write_file returns WRITE_OK, continue only if additional edits are needed; otherwise answer done.\n" +
			"- If the user asks to implement or fix, identify target files and exact modifications first.\n" +
			"- Never expose tool-call syntax, DSML, XML-like invoke blocks, or raw file payloads in the final answer.\n" +
			"- Keep final answers concise and actionable.\n\n" +
			"Project context:\n" + systemContext,
	}
}

func buildCodingTools() []Tool {
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
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "write_file",
				Description: "Writes full file content to a relative path inside the selected project.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Relative file path to write. Example: README.md",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Full final content to write into the file.",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDefinition{
				Name:        "run_command",
				Description: "Runs a shell command inside the selected project directory and returns stdout/stderr + exit code.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "Shell command to execute in project root. Example: flutter create mobile_app",
						},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}
