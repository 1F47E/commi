package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	antModel         = "claude-3-5-sonnet-20240620"
	openaiModel      = "gpt-4"
	maxTokens        = 4000
	llmClientTimeout = 30 * time.Second
)

func truncatePrompt(prompt string, maxTokens int) string {
	promptTokens := len(strings.Split(prompt, " ")) * 2

	log.Debug().Msg(fmt.Sprintf("Prompt tokens: %d", promptTokens))
	if promptTokens > maxTokens {
		words := strings.Split(prompt, " ")
		truncatedWords := words[:maxTokens/2]
		prompt = strings.Join(truncatedWords, " ")
		prompt += "..."
		log.Warn().Msg(fmt.Sprintf("Truncating prompt to %d tokens", maxTokens))
	}

	return prompt
}

type LLMClient interface {
	GenerateCommitMessage(status, diffs, subject string) (string, error)
}

// OpenAI Client

type OpenAIClient struct {
	apiKey string
	url    string
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		url:    "https://api.openai.com/v1/chat/completions",
	}
}

func (c *OpenAIClient) GenerateCommitMessage(status, diffs, subject string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", status, diffs)
	if subject != "" {
		prompt += fmt.Sprintf("\n\nPlease focus on the following subject in your commit message: %s", subject)
	}
	prompt = truncatePrompt(prompt, maxTokens)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      openaiModel,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": SystemPrompt,
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

	req, err := http.NewRequest("POST", c.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{
		Timeout: llmClientTimeout,
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

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	return content, nil
}

// Anthropic Client

type AnthropicClient struct {
	apiKey string
	url    string
}

func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		apiKey: apiKey,
		url:    "https://api.anthropic.com/v1/messages",
	}
}

func (c *AnthropicClient) GenerateCommitMessage(status, diffs, subject string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message in XML format:", status, diffs)
	if subject != "" {
		prompt += fmt.Sprintf("\n\nPlease focus on the following subject in your commit message: %s", subject)
	}
	prompt = truncatePrompt(prompt, maxTokens)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      antModel,
		"max_tokens": maxTokens,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": prompt},
				},
			},
		},
		"system": SystemPrompt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", c.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: llmClientTimeout,
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
	log.Debug().Msg(fmt.Sprintf("Response body:\n%s", string(body)))

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

	text := result.Content[0].Text
	log.Debug().Msg(fmt.Sprintf("Response text:\n%s", text))

	return text, nil
}
