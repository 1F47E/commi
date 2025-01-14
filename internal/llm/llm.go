package llm

import "time"

const (
	MaxTokens        = 4000
	LLMClientTimeout = 30 * time.Second

	// Provider types
	ProviderAnthropic = "ANTHROPIC"
	ProviderOpenAI    = "OPENAI"
)

func ValidateProvider(provider string) bool {
	switch provider {
	case ProviderAnthropic, ProviderOpenAI:
		return true
	default:
		return false
	}
}

const SystemPrompt = `You are an AI assistant that helps developers write better commit messages. Your task is to analyze the git status and diffs, and generate a descriptive and informative commit message that follows best practices.

Please follow these guidelines:
• Keep the title concise (max 72 characters) but descriptive
• Use the imperative mood ("Add feature" not "Added feature")
• Start with a capital letter
• Don't end the title with a period
• Provide a detailed description when the changes are complex
• Break down the description into bullet points for multiple changes
• Reference any relevant issue numbers

Format your response in XML with the following structure:
<commit>
  <title>Your concise title here</title>
  <description>
    Your detailed description here (optional)
  </description>
</commit>`

type LLMProvider interface {
	GenerateCommitMessage(systemPrompt, status, diffs, subject string) (string, error)
}

func TruncatePrompt(prompt string, maxTokens int) string {
	if len(prompt) > maxTokens {
		return prompt[:maxTokens] + "..."
	}
	return prompt
}
