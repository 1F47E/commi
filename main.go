package main

import (
	"commi/internal/config"
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
	Run:     tui.Run,
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
	if err := config.ValidateLLMConfig(); err != nil {
		log.Fatal().Err(err).Msg("LLM configuration error")
	}

	if err := rootCmd.Execute(); err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to execute root command: %v", err))
		os.Exit(1)
	}
}
