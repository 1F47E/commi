package config

import (
	"fmt"
	"os"
)

type LLMConfig struct {
	APIKey string
	Model  string
}

func ValidateLLMConfig() error {
	if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("OPENAI_API_KEY") == "" {
		return fmt.Errorf("no LLM API key found. Please set either ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable")
	}
	return nil
}
