package anthropic

import (
	"bytes"
	"commi/internal/config"
	"commi/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	MaxTokensOutput  = 4096
	MaxTokensInput   = 10000 // TODO: make this configurable and implement limit
	LLMClientTimeout = 10 * time.Second
)

const (
	anthropicVersion = "2023-06-01"
	defaultModel     = "claude-3-sonnet-20240229"
	apiURL           = "https://api.anthropic.com/v1/messages"
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

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
	Type string `json:"type"`
}

func (c *AnthropicClient) GenerateCommitMessage(ctx context.Context, sysPrompt, status, diffs, subject string) (string, error) {
	// Combine system prompt and user prompt into a single message
	prompt := fmt.Sprintf("%s\n\nGit status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", sysPrompt, status, diffs)
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
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	client := &http.Client{
		Timeout: LLMClientTimeout,
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

	if utils.IsDebug() {
		// http code
		log.Debug().Msgf("Anthropic response: %s", string(body))
	}

	var response anthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Check for API error response
	if response.Type == "error" && response.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", response.Error.Type, response.Error.Message)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response: %s", string(body))
	}

	return response.Content[0].Text, nil
}
