package main

import (
	"commi/internal/clients/anthropic"
	"commi/internal/clients/openai"
	"commi/internal/config"
	"commi/internal/core"
	"commi/internal/tui"
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
	Use:     "commi [subject]",
	Short:   "Generate and apply AI-powered commit messages",
	Run:     runCommand,
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

func getProvider() (core.LLMClient, error) {
	// TODO: move to config, add selector for models
	// Try to initialize Anthropic client first
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		log.Debug().Msg("Using Anthropic as LLM provider")
		return anthropic.NewAnthropicClient(config.LLMConfig{APIKey: key}), nil
	}

	// Try OpenAI client second
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		log.Debug().Msg("Using OpenAI as LLM provider")
		return openai.NewOpenAIClient(config.LLMConfig{APIKey: key}), nil
	}

	return nil, fmt.Errorf("no LLM providers available. Please set either ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable")
}

func runCommand(cmd *cobra.Command, args []string) {
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Println(cmd.Version)
		return
	}

	provider, err := getProvider()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize LLM provider")
	}

	c := core.NewCore(provider)
	tui.Run(cmd, args, c)
}

func main() {
	if err := config.ValidateLLMConfig(); err != nil {
		log.Fatal().Err(err).Msg("LLM configuration error")
	}

	if err := rootCmd.Execute(); err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to execute root command: %v", err))
		os.Exit(1)
	}
}
