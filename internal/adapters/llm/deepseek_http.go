package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const deepSeekRequestTimeout = 90 * time.Second

func (c *DeepSeekClient) doChatRequest(messages []Message, tools []Tool) (Message, error) {
	reqBody := RequestBody{
		Model:    deepSeekModel,
		Messages: messages,
		Tools:    tools,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), deepSeekRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", deepSeekURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: deepSeekRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		if errorsIsTimeout(err) {
			return Message{}, fmt.Errorf("DeepSeek request timed out after %s", deepSeekRequestTimeout)
		}
		return Message{}, err
	}

	bodyText, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return Message{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("API error: %s", string(bodyText))
	}

	var respBody ResponseBody
	if err := json.Unmarshal(bodyText, &respBody); err != nil {
		return Message{}, err
	}
	if len(respBody.Choices) == 0 {
		return Message{}, fmt.Errorf("no response from DeepSeek API")
	}

	return respBody.Choices[0].Message, nil
}

func errorsIsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
