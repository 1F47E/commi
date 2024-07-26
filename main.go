package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// ===== CONSTANTS

// version is set during build
var version string

// errorLoggingOnly is set during build
var errorLoggingOnly string

// ===== ROOT COMMAND

var rootCmd = &cobra.Command{
	Use:     "aicommit",
	Short:   "Generate and apply AI-powered commit messages",
	Run:     runAICommit,
	Version: version,
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Display version information")

	// Configure zerolog
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "",
		NoColor:    false,
	}
	
	// Set log level based on the errorLoggingOnly flag
	if errorLoggingOnly == "true" {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	
	log.Logger = log.Output(output)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to execute root command: %v", err))
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

// LLMClient interface is defined in clients.go

func generateCommitMessage(client LLMClient, status, diffs string) (*commit, error) {
	modelName := getModelName(client)
	log.Debug().Str("model", modelName).Msg("Using AI model for commit message generation")

	spinner := NewSpinner()
	spinner.Start("Generating commit message...")

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
