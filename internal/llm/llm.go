package llm

import "time"

const (
	MaxTokensOutput  = 5000
	MaxTokensInput   = 10000 // TODO: make this configurable and implement limit
	LLMClientTimeout = 10 * time.Second
)

type LLMProviderType string

const (
	LLMProviderTypeAnthropic LLMProviderType = "ANTHROPIC"
	LLMProviderTypeOpenAI    LLMProviderType = "OPENAI"
)

func ValidateProvider(provider LLMProviderType) bool {
	switch provider {
	case LLMProviderTypeAnthropic, LLMProviderTypeOpenAI:
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
  <title>Your title here</title>
  <description>
    Your detailed description here
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
