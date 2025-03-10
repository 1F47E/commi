package openai

import (
	"commi/internal/clients/common"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	MaxTokensOutput = 5000
	MaxTokensInput  = 10000 // TODO: make this configurable and implement limit
)

const (
	// defaultModel = "o3-mini"
	defaultModel = "gpt-4o-mini"
	apiURL       = "https://api.openai.com/v1/chat/completions"
)

type OpenAIClient struct {
	apiKey string
	model  string
	client *http.Client
	config common.ClientConfig
}

func NewOpenAIClient(key string) *OpenAIClient {

	clientConfig := common.DefaultConfig()
	clientConfig.Headers = map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + key,
	}

	return &OpenAIClient{
		apiKey: key,
		model:  defaultModel,
		client: common.NewHTTPClient(clientConfig),
		config: clientConfig,
	}
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (c *OpenAIClient) handleResponse(resp *http.Response) (*openaiResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var response openaiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", response.Error.Type, response.Error.Message)
	}

	return &response, nil
}

func (c *OpenAIClient) GenerateCommitMessage(ctx context.Context, sysPrompt, status, diffs, subject string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", status, diffs)
	if subject != "" {
		prompt += fmt.Sprintf("\n\nPlease focus on the following subject in your commit message: %s", subject)
	}

	if len(prompt) > MaxTokensInput {
		prompt = prompt[:MaxTokensInput]
	}
	requestBody, err := json.Marshal(map[string]interface{}{
		"model": c.model,
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
		"max_tokens": MaxTokensOutput,
		// "max_completion_tokens": MaxTokensOutput, // TODO: support gpt models with max_tokens
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := common.NewRequest(http.MethodPost, apiURL, requestBody, c.config)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	response, err := c.handleResponse(resp)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}
