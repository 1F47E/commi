package openai

import (
	"bytes"
	"commi/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	MaxTokensOutput = 5000
	MaxTokensInput  = 10000 // TODO: make this configurable and implement limit
	timeout         = 10 * time.Second
)

const (
	defaultModel = "gpt-4"
	apiURL       = "https://api.openai.com/v1/chat/completions"
)

type OpenAIClient struct {
	apiKey string
	model  string
}

func NewOpenAIClient(config config.LLMConfig) *OpenAIClient {
	model := config.Model
	if model == "" {
		model = defaultModel
	}
	return &OpenAIClient{
		apiKey: config.APIKey,
		model:  model,
	}
}

func (c *OpenAIClient) GenerateCommitMessage(ctx context.Context, sysPrompt, status, diffs, subject string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", status, diffs)
	if subject != "" {
		prompt += fmt.Sprintf("\n\nPlease focus on the following subject in your commit message: %s", subject)
	}

	// Truncate prompt if too long
	if len(prompt) > MaxTokensInput {
		prompt = prompt[:MaxTokensInput]
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      c.model,
		"max_tokens": MaxTokensOutput,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": sysPrompt,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{
		Timeout: timeout,
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

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}
