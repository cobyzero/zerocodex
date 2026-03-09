package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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

	req, err := http.NewRequest("POST", deepSeekURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
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
