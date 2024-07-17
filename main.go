package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ===== CONSTANTS

// version is set during build
var version string

// ===== ROOT COMMAND

var rootCmd = &cobra.Command{
	Use:     "aicommit",
	Short:   "Generate and apply AI-powered commit messages",
	Run:     runAICommit,
	Version: version,
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Display version information")
}

func main() {

	if err := rootCmd.Execute(); err != nil {
		log.Error().Err(err).Msg("Failed to execute root command")
		os.Exit(1)
	}
}

// ===== AI COMMIT GENERATION

func runAICommit(cmd *cobra.Command, args []string) {
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Println(version)
		return
	}
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
	modelName := getModelName(client)
	spinner := NewSpinner()
	spinner.Start(fmt.Sprintf("Generating commit message using %s...", modelName))

	commitMessage, err := client.GenerateCommitMessage(status, diffs)

	spinner.Stop()

	if err != nil {
		return nil, err
	}

	return commitMessage, nil
}

func getModelName(client LLMClient) string {
	switch client.(type) {
	case *AnthropicClient:
		return "Anthropic Claude"
	case *OpenAIClient:
		return "OpenAI GPT-4"
	default:
		return "Unknown Model"
	}
}
