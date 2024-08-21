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
	rootCmd.Flags().BoolP("force", "f", false, "Force commit without showing the menu")
	rootCmd.Flags().StringP("prefix", "p", "", "Specify a custom commit message prefix")

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

	forceFlag, _ := cmd.Flags().GetBool("force")
	prefix, _ := cmd.Flags().GetString("prefix")
	log.Debug().Msgf("Force: %t", forceFlag)
	log.Debug().Msgf("Prefix: %s", prefix)

	commitMessage, err := generateCommitMessage(client, status, diffs, subject)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate commit message")
		os.Exit(1)
	}

	if prefix != "" {
		commitMessage.Title = prefix + " " + commitMessage.Title
	}

	if forceFlag {
		handleForcedCommit(commitMessage)
	} else {
		handleUserResponse(cmd, args, commitMessage)
	}
}

func handleForcedCommit(commitMessage *commit.Commit) {
	err := applyCommit(commitMessage)
	if err != nil {
		log.Error().Err(err).Msg("Failed to apply commit")
		os.Exit(1)
	}
	fmt.Printf("Commit applied: %s\n", commitMessage.Title)
	os.Exit(0)
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
	spinner := NewSpinner()
	spinner.Start("Generating commit message...")

	sys := llm.SystemPrompt
	if _, exists := os.LookupEnv("DISABLE_EMOJI"); !exists {
		sys += "\n• Please follow the gitmoji standard (https://gitmoji.dev/) and feel free to use emojis in the commit messages where appropriate to enhance readability and convey the nature of the changes."
	}

	xmlContent, err := client.GenerateCommitMessage(sys, status, diffs, subject)

	spinner.Stop()

	if err != nil {
		return nil, err
	}

	return xmlparser.ParseXMLCommit(xmlContent)
}

func applyCommit(c *commit.Commit) error {
	// Stage all changes
	stageCmd := exec.Command("git", "add", "-A")
	stageOutput, stageErr := stageCmd.CombinedOutput()
	if stageErr != nil {
		return fmt.Errorf("failed to stage changes: %v\nOutput: %s", stageErr, string(stageOutput))
	}

	// Commit the staged changes
	message := c.Title
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitOutput, commitErr := commitCmd.CombinedOutput()
	if commitErr != nil {
		return fmt.Errorf("failed to apply commit: %v\nOutput: %s", commitErr, string(commitOutput))
	}
	return nil
}
