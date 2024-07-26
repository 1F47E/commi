package main

import (
	"commi/commit"
	"commi/llm"
	"commi/xmlparser"
	"fmt"
	"os"
	"os/exec"

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
	Use:     "aicommit [subject]",
	Short:   "Generate and apply AI-powered commit messages",
	Run:     runAICommit,
	Version: version,
	Args:    cobra.MaximumNArgs(1),
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Display version information")
	rootCmd.Flags().BoolP("auto", "a", false, "Automatically commit without dialog")

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

	var subject string
	if len(args) > 0 {
		subject = args[0]
	}

	commitMessage, err := generateCommitMessage(client, status, diffs, subject)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate commit message")
		os.Exit(1)
	}

	autoFlag, _ := cmd.Flags().GetBool("auto")
	if autoFlag {
		err = applyCommit(commitMessage)
		if err != nil {
			log.Error().Err(err).Msg("Failed to apply commit")
			os.Exit(1)
		}
		log.Info().Msg("Commit applied automatically")
	} else {
		handleUserResponse(cmd, args, commitMessage)
	}
}

func getClient() (llm.LLMClient, error) {
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey != "" {
		return llm.NewAnthropicClient(anthropicKey), nil
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey != "" {
		return llm.NewOpenAIClient(openaiKey), nil
	}

	return nil, fmt.Errorf("no API key found for Anthropic or OpenAI")
}

func generateCommitMessage(client llm.LLMClient, status, diffs, subject string) (*commit.Commit, error) {
	modelName := getModelName(client)
	log.Debug().Str("model", modelName).Msg("Using AI model for commit message generation")

	spinner := NewSpinner()
	spinner.Start("Generating commit message...")

	xmlContent, err := client.GenerateCommitMessage(status, diffs, subject)

	spinner.Stop()

	if err != nil {
		return nil, err
	}

	return xmlparser.ParseXMLCommit(xmlContent)
}

func getModelName(client llm.LLMClient) string {
	switch client.(type) {
	case *llm.AnthropicClient:
		return "Anthropic Claude"
	case *llm.OpenAIClient:
		return "OpenAI GPT-4"
	default:
		return "Unknown Model"
	}
}

func applyCommit(c *commit.Commit) error {
	message := c.Title
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply commit: %v\nOutput: %s", err, string(output))
	}
	return nil
}
