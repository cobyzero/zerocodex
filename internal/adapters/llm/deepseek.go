package llm

type DeepSeekClient struct {
	APIKey string
}

func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	return &DeepSeekClient{
		APIKey: apiKey,
	}
}

const (
	deepSeekURL      = "https://api.deepseek.com/chat/completions"
	deepSeekModel    = "deepseek-chat"
	maxToolIterations = 10
)
