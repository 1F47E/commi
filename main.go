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

const (
	antModel     = "claude-3-5-sonnet-20240620"
	openaiModel  = "gpt-4o"
	maxTokens    = 4000
	systemPrompt = `
	You are an AI assistant specialized in generating descriptive git commit messages. 
	Analyze the provided git status and diff, then create a commit message that accurately summarizes the changes. 
	Focus on the most important modifications and their impact. Keep the message clear and to the point. NO YAPPING.
	If possible use file path and describe changes in each file or dir. Use multiline style with bullet points. You are allowed to use emoji but not excessive.
	`
)

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

type model struct {
	list          list.Model
	commitMessage string
	choice        string
}

func initialModel(commitMessage string) model {
	items := []list.Item{
		item{title: "Yes", desc: "Proceed with this commit message"},
		item{title: "No", desc: "Abort the commit"},
		item{title: "Redo", desc: "Regenerate the commit message"},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Do you want to proceed with this commit message?"

	return model{
		list:          l,
		commitMessage: commitMessage,
	}
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = i.title
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(2, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
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

func runAICommit(cmd *cobra.Command, args []string) {
	status, diff, err := getGitInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get git information")
		os.Exit(1)
	}

	apiKey := getAPIKey()
	if apiKey == "" {
		log.Error().Msg("ANTHROPIC_API_KEY environment variable is not set")
		os.Exit(1)
	}

	client := NewAnthropicClient(apiKey)
	commitMessage, err := generateCommitMessage(client, status, diff)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate commit message")
		os.Exit(1)
	}

	handleUserResponse(cmd, args, commitMessage)
}

func getGitInfo() (string, string, error) {
	status, err := getGitStatus()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git status: %w", err)
	}

	diff, err := getGitDiff()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git diff: %w", err)
	}

	return status, diff, nil
}

func getAPIKey() string {
	return os.Getenv("ANTHROPIC_API_KEY")
}

func generateCommitMessage(client *AnthropicClient, status, diff string) (string, error) {
	commitMessage, err := client.GenerateCommitMessage(status, diff)
	if err != nil {
		return "", err
	}

	log.Info().Msg("Generated commit message:")
	fmt.Println(commitMessage)

	return commitMessage, nil
}

func handleUserResponse(cmd *cobra.Command, args []string, commitMessage string) {
	p := tea.NewProgram(initialModel(commitMessage))
	m, err := p.Run()
	if err != nil {
		log.Error().Err(err).Msg("Error running Bubble Tea program")
		os.Exit(1)
	}

	if m, ok := m.(model); ok {
		switch m.choice {
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

func (c *OpenAIClient) GenerateCommitMessage(status, diff string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diff:\n\n%s\n\nBased on this information, generate a good and descriptive commit message, fit into 100 tokens max:", status, diff)

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

func (c *AnthropicClient) GenerateCommitMessage(status, diff string) (string, error) {
	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diff:\n\n%s\n\n", status, diff)
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

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "--no-pager", "diff")
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
