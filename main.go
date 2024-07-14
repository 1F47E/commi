package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const systemPrompt = `
# Commit Message Generator

You are an AI assistant designed to generate high-quality commit messages in the style of a senior software engineer. 
You will be provided with the output of "git status" and "git diff" for each modified file. 
Your task is to analyze this information and create a concise, informative, and professional commit message.

## Guidelines for Generating Commit Messages:

1. Start with a brief, descriptive summary (50 characters or less) that captures the essence of the change.
2. Use the imperative mood in the summary line (e.g., "Add feature" not "Added feature" or "Adds feature").
3. Provide more detailed explanations in subsequent lines, if necessary, wrapping at 72 characters.
4. Focus on explaining the "why" behind the changes, not just the "what".
5. Mention any breaking changes prominently.
6. Reference relevant issue numbers or ticket IDs if applicable.
7. Use a consistent style and terminology.
8. Avoid redundant information that's already in the diff.
9. Separate the summary from the body with a blank line.
10. Use bullet points for multiple items in the body.

## Example Structure:

---
Implement user authentication system

- Add JWT-based authentication middleware
- Create login and registration endpoints
- Update user model to include password hashing
- Integrate with frontend login form
---

## Additional Considerations:

- If multiple files are changed, try to summarize the overall impact rather than listing each file.
- If the changes are part of a larger feature or refactoring effort, mention this context.
- For bug fixes, briefly describe the bug and how the change addresses it.
- Use technical language appropriate for the codebase, but avoid excessive jargon.
- If the commit includes both functional changes and code style improvements, prioritize describing the functional changes.

Analyze the provided git status and diff information, and generate a commit message that adheres to these guidelines and reflects the work of a senior engineer.
`

const (
	antModel    = "claude-3-5-sonnet-20240620"
	openaiModel = "gpt-4o"
	maxTokens   = 4000
	// systemPrompt = `
	// You are an AI assistant specialized in generating descriptive git commit messages.
	// Analyze the provided git status and diff, then create a commit message that accurately summarizes the changes.
	// Focus on the most important modifications and their impact. Keep the message clear and to the point. NO YAPPING.
	// You are allowed to use emoji but not excessive.
	// `
)

// If possible use file path and describe changes in each file or dir. Use multiline style with bullet points. You are allowed to use emoji but not excessive.

var rootCmd = &cobra.Command{
	Use:   "aicommit",
	Short: "Generate and apply AI-powered commit messages",
	Run:   runAICommit,
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("Failed to execute root command")
		os.Exit(1)
	}
}

func runAICommit(cmd *cobra.Command, args []string) {
	status, diffs, err := getGitInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get git information")
		os.Exit(1)
	}

	client, err := getClient()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize AI client")
	}

	commitMessage, err := generateCommitMessage(client, status, diffs)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate commit message")
		os.Exit(1)
	}

	handleUserResponse(cmd, args, commitMessage)
}

func getClient() (LLMClient, error) {
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey != "" {
		return NewAnthropicClient(anthropicKey), nil
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey != "" {
		return NewOpenAIClient(openaiKey), nil
	}

	return nil, fmt.Errorf("no API key found for Anthropic or OpenAI")
}

type LLMClient interface {
	GenerateCommitMessage(status, diffs string) (string, error)
}

func generateCommitMessage(client LLMClient, status, diffs string) (string, error) {
	commitMessage, err := client.GenerateCommitMessage(status, diffs)
	if err != nil {
		return "", err
	}

	// log.Info().Msg("Generated commit message:")
	// fmt.Println(commitMessage)

	return commitMessage, nil
}

// ===== TUI

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list          list.Model
	commitMessage string
	choice        string
	quitting      bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"Commit Message:\n\n%s\n\n%s",
		m.commitMessage,
		m.list.View(),
	)
}

func handleUserResponse(cmd *cobra.Command, args []string, commitMessage string) {
	items := []list.Item{
		item("Yes"),
		item("No"),
		item("Redo"),
	}

	const defaultWidth = 30

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Do you want to proceed with this commit message?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{list: l, commitMessage: commitMessage}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
		os.Exit(1)
	}

	if finalModel, ok := finalModel.(model); ok {
		switch finalModel.choice {
		case "No":
			log.Info().Msg("Commit aborted.")
		case "Redo":
			log.Info().Msg("Regenerating commit message...")
			runAICommit(cmd, args)
		case "Yes":
			err := executeGitCommit(commitMessage)
			if err != nil {
				log.Error().Err(err).Msg("Failed to execute git commit")
				os.Exit(1)
			}
			log.Info().Msg("Commit successfully created!")
		}
	}
}

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

func (c *OpenAIClient) GenerateCommitMessage(status, diffs string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\nBased on this information, generate a good and descriptive commit message, fit into 100 tokens max:", status, diffs)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      openaiModel,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
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

	client := &http.Client{}
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

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

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

func (c *AnthropicClient) GenerateCommitMessage(status, diffs string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diffs:\n\n%s\n\n", status, diffs)
	// Calculate token count for the prompt
	promptTokens := len(strings.Split(prompt, " ")) * 2

	log.Info().Msgf("Prompt tokens: %d", promptTokens)
	// Truncate the prompt if it exceeds maxTokens
	if promptTokens > maxTokens {
		words := strings.Split(prompt, " ")
		truncatedWords := words[:maxTokens/2]
		prompt = strings.Join(truncatedWords, " ")
		prompt += "..."
		log.Info().Msgf("Truncated prompt len: %d", len(prompt))
	}

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
		"system": systemPrompt,
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

	client := &http.Client{}
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

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("unexpected response format")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected content format")
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		return "", fmt.Errorf("text not found in response")
	}

	return strings.TrimSpace(text), nil
}

func getGitInfo() (string, string, error) {
	status, err := getGitStatus()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git status: %w", err)
	}

	files, err := getChangedFiles(status)
	if err != nil {
		return "", "", fmt.Errorf("failed to get changed files: %w", err)
	}

	diffs := ""
	for _, file := range files {
		diff, err := getGitDiff(file)
		if err != nil {
			log.Warn().Err(err).Str("file", file).Msg("Failed to get diff for file")
			continue
		}
		diffs += fmt.Sprintf("Diff for %s:\n%s\n\n", file, diff)
	}

	return status, diffs, nil
}

func getGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if strings.Contains(string(output), "nothing to commit") {
		return "", fmt.Errorf("nothing to commit")
	}
	return string(output), nil
}

func getChangedFiles(status string) ([]string, error) {
	lines := strings.Split(status, "\n")
	var files []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return nil, fmt.Errorf("unexpected git status output format")
		}
		files = append(files, parts[1])
	}
	return files, nil
}

func getGitDiff(file string) (string, error) {
	cmd := exec.Command("git", "--no-pager", "diff", file)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func executeGitCommit(message string) error {
	cmd := exec.Command("git", "commit", "-am", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %v\nOutput: %s", err, string(output))
	}
	log.Info().Msg("Git commit executed successfully")
	return nil
}
