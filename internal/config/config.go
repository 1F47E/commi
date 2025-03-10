package config

import (
	"context"
	"fmt"
	"os"
)

type LLMConfig struct {
	APIKey string
	Model  string
}

// LLMProvider defines the interface for LLM clients
type LLMProvider interface {
	GenerateCommitMessage(ctx context.Context, systemPrompt, status, diffs, subject string) (string, error)
}

func ValidateLLMConfig() error {
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("OPENAI_API_KEY") == "" {
		return fmt.Errorf("no LLM API key found. Please set either ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable")
	}
	return nil
}
