package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ===== CONSTANTS

const (
	antModel    = "claude-3-5-sonnet-20240620"
	openaiModel = "gpt-4"
	maxTokens   = 4000
)

// ===== ROOT COMMAND

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

// ===== AI COMMIT GENERATION

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

type commit struct {
	Title   string
	Message string
}

// LLMClient interface is defined in clients.go

func generateCommitMessage(client LLMClient, status, diffs string) (*commit, error) {
	spinner := initializeSpinner()
	spinnerProgram, spinnerDone := runSpinner(spinner)

	commitMessage, err := client.GenerateCommitMessage(status, diffs)

	stopSpinner(spinnerProgram, spinnerDone)

	if err != nil {
		return nil, err
	}

	return commitMessage, nil
}
