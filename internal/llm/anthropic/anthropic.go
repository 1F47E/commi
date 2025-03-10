package anthropic

import (
	"bytes"
	"commi/internal/config"
	"commi/internal/llm"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultModel = "claude-3-5-sonnet-20241022"
	apiURL       = "https://api.anthropic.com/v1/messages"
)

type AnthropicClient struct {
	apiKey string
	model  string
}

func NewAnthropicClient(config config.LLMConfig) *AnthropicClient {
	model := config.Model
	if model == "" {
		model = defaultModel
	}
	return &AnthropicClient{
		apiKey: config.APIKey,
		model:  model,
	}
}

func (c *AnthropicClient) GenerateCommitMessage(sysPrompt, status, diffs, subject string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", status, diffs)
	if subject != "" {
		prompt += fmt.Sprintf("\n\nPlease focus on the following subject in your commit message: %s", subject)
	}
	prompt = llm.TruncatePrompt(prompt, llm.MaxTokensInput)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      c.model,
		"max_tokens": llm.MaxTokensOutput,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
		"system": sysPrompt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: llm.LLMClientTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("unexpected response format")
	}

	return result.Content[0].Text, nil
}
