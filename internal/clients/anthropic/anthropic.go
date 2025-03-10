package anthropic

import (
	"commi/internal/clients/common"
	"commi/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

const (
	MaxTokensOutput = 4096
	MaxTokensInput  = 10000 // TODO: make this configurable and implement limit
)

const (
	anthropicVersion = "2023-06-01"
	// defaultModel     = "claude-3-sonnet-20240229"
	// defaultModel     = "claude-3-5-sonnet-20240620"
	defaultModel = "claude-3-7-sonnet-20250219"
	apiURL       = "https://api.anthropic.com/v1/messages"
)

type AnthropicClient struct {
	apiKey string
	model  string
	client *http.Client
	config common.ClientConfig
}

func NewAnthropicClient(key string) *AnthropicClient {

	clientConfig := common.DefaultConfig()
	clientConfig.Headers = map[string]string{
		"Content-Type":      "application/json",
		"x-api-key":         key,
		"anthropic-version": anthropicVersion,
	}

	return &AnthropicClient{
		apiKey: key,
		model:  defaultModel,
		client: common.NewHTTPClient(clientConfig),
		config: clientConfig,
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

func (c *AnthropicClient) handleResponse(resp *http.Response) (*anthropicResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if utils.IsDebug() {
		log.Debug().Msgf("Anthropic response status: %d", resp.StatusCode)
		log.Debug().Msgf("Anthropic response body: %s", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var response anthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Type == "error" && response.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", response.Error.Type, response.Error.Message)
	}

	return &response, nil
}

func (c *AnthropicClient) GenerateCommitMessage(ctx context.Context, sysPrompt, status, diffs, subject string) (string, error) {
	prompt := fmt.Sprintf("%s\n\nGit status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", sysPrompt, status, diffs)
	if subject != "" {
		prompt += fmt.Sprintf("\n\nPlease focus on the following subject in your commit message: %s", subject)
	}

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

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return response.Content[0].Text, nil
}
