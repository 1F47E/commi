package config

import (
	"commi/internal/llm"
	"fmt"
	"os"
)

type LLMConfig struct {
	APIKey string
	Model  string
}

func ValidateLLMConfig() error {
	provider := os.Getenv("LLM_PROVIDER")
	if provider != "" {
		providerType := llm.LLMProviderType(provider)
		if !llm.ValidateProvider(providerType) {
			return fmt.Errorf("invalid LLM_PROVIDER value: %q. Must be either %s or %s",
				provider, llm.LLMProviderTypeAnthropic, llm.LLMProviderTypeOpenAI)
		}

		// If provider is specified, its API key must be set
		switch providerType {
		case llm.LLMProviderTypeAnthropic:
			if os.Getenv("ANTHROPIC_API_KEY") == "" {
				return fmt.Errorf("%s selected as provider but ANTHROPIC_API_KEY is not set", llm.LLMProviderTypeAnthropic)
			}
		case llm.LLMProviderTypeOpenAI:
			if os.Getenv("OPENAI_API_KEY") == "" {
				return fmt.Errorf("%s selected as provider but OPENAI_API_KEY is not set", llm.LLMProviderTypeOpenAI)
			}
		}
	} else {
		// If no provider specified, at least one API key must be set
		if os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("OPENAI_API_KEY") == "" {
			return fmt.Errorf("no LLM providers available. Please set either ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable")
		}
	}
	return nil
}
