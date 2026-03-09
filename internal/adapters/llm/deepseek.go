package llm

import (
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

const (
	deepSeekURL      = "https://api.deepseek.com/chat/completions"
	deepSeekModel    = "deepseek-chat"
	maxToolIterations = 10
)
