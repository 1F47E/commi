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

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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

func runAICommit(cmd *cobra.Command, args []string) {
	status, err := getGitStatus()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get git status")
		os.Exit(1)
	}

	diff, err := getGitDiff()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get git diff")
		os.Exit(1)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Error().Msg("ANTHROPIC_API_KEY environment variable is not set")
		os.Exit(1)
	}

	commitMessage, err := generateCommitMessage(status, diff, apiKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate commit message")
		os.Exit(1)
	}

	log.Info().Msg("Generated commit message:")
	fmt.Println(commitMessage)

	err = executeGitCommit(commitMessage)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute git commit")
		os.Exit(1)
	}

	log.Info().Msg("Commit successfully created!")
}

func getGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

const (
	modelVersion = "claude-3-5-sonnet-20240620"
	maxTokens    = 3000
	systemPrompt = `You are an AI assistant specialized in generating concise and descriptive git commit messages. Analyze the provided git status and diff, then create a commit message that accurately summarizes the changes. Focus on the most important modifications and their impact. Keep the message clear and to the point. NO YAPPING.`
)

func getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	if len(output) > maxTokens {
		log.Warn().Msg("Git diff truncated due to length")
		output = output[:maxTokens]
		output = append(output, []byte("\n... (truncated)")...)
	}

	return string(output), nil
}

func generateCommitMessage(status, diff, apiKey string) (string, error) {
	url := "https://api.anthropic.com/v1/messages"

	prompt := fmt.Sprintf("Git status:\n\n%s\n\nGit diff:\n\n%s\n\nBased on this information, generate a concise and descriptive commit message:", status, diff)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":      modelVersion,
		"max_tokens": 1000,
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

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
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

func executeGitCommit(message string) error {
	log.Info().Msgf("Executing git commit with message: \n\n%s", message)
	return nil

	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %v\nOutput: %s", err, string(output))
	}
	log.Info().Msg("Git commit executed successfully")
	return nil
}
