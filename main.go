package main

import (
	"commi/internal/clients/anthropic"
	"commi/internal/clients/openai"
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
	// Initialize available providers
	providers := make(map[string]core.LLMClient)

	// Check for Anthropic API key
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		log.Debug().Msg("Found Anthropic API key")
		providers["ANTHROPIC"] = anthropic.NewAnthropicClient(key)
	}

	// Check for OpenAI API key
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		log.Debug().Msg("Found OpenAI API key")
		providers["OPENAI"] = openai.NewOpenAIClient(key)
	}

	// If no providers available, return error
	if len(providers) == 0 {
		return nil, fmt.Errorf("no LLM providers available. Please set either ANTHROPIC_API_KEY or OPENAI_API_KEY environment variable")
	}

	// Check if provider is explicitly set via environment variable
	if provider := os.Getenv("COMMI_LLM_PROVIDER"); provider != "" {
		if client, exists := providers[provider]; exists {
			log.Debug().Msgf("Using %s as LLM provider (from COMMI_LLM_PROVIDER)", provider)
			return client, nil
		}
		log.Warn().Msgf("Provider %s specified in COMMI_LLM_PROVIDER not available, falling back to auto-detection", provider)
	}

	// If only one provider available, use it
	if len(providers) == 1 {
		for name, client := range providers {
			log.Debug().Msgf("Using %s as LLM provider (only one available)", name)
			return client, nil
		}
	}

	// Prefer Anthropic if available
	if client, exists := providers["ANTHROPIC"]; exists {
		log.Debug().Msg("Using Anthropic as LLM provider (preferred)")
		return client, nil
	}

	// Otherwise use OpenAI
	log.Debug().Msg("Using OpenAI as LLM provider")
	return providers["OPENAI"], nil
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
	if err := rootCmd.Execute(); err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to execute root command: %v", err))
		os.Exit(1)
	}
}
