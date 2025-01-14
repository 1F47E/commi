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
		if !llm.ValidateProvider(provider) {
			return fmt.Errorf("invalid LLM_PROVIDER value: %q. Must be either %s or %s",
				provider, llm.ProviderAnthropic, llm.ProviderOpenAI)
		}

		// If provider is specified, its API key must be set
		switch provider {
		case llm.ProviderAnthropic:
			if os.Getenv("ANTHROPIC_API_KEY") == "" {
				return fmt.Errorf("%s selected as provider but ANTHROPIC_API_KEY is not set", llm.ProviderAnthropic)
			}
		case llm.ProviderOpenAI:
			if os.Getenv("OPENAI_API_KEY") == "" {
				return fmt.Errorf("%s selected as provider but OPENAI_API_KEY is not set", llm.ProviderOpenAI)
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
