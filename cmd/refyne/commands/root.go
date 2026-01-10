// Package commands implements the CLI commands for refyne.
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "refyne",
	Short: "LLM-powered web scraper for structured data extraction",
	Long: `Refyne extracts structured data from web pages using LLMs.

Define a schema for the data you want, point it at URLs, and get
validated, structured output in JSON, JSONL, or YAML.

Examples:
  # Extract data from a single page
  refyne scrape -u "https://example.com/page" -s schema.json

  # Crawl and extract from multiple pages
  refyne scrape -u "https://example.com/search" -s schema.json \
      --follow "a.item-link" --max-depth 1

  # Use OpenRouter with a specific model
  refyne scrape -u "https://example.com" -s schema.yaml \
      -p openrouter -m anthropic/claude-sonnet

  # Use local Ollama
  refyne scrape -u "https://example.com" -s schema.json \
      -p ollama -m llama3.2`,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().String("config", "", "config file (default $HOME/.refyne.yaml)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress progress output")

	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
}

func initConfig() {
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.AddConfigPath(".")
		viper.SetConfigName(".refyne")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("REFYNE")
	viper.AutomaticEnv()

	// Also check common API key env vars
	_ = viper.BindEnv("api_key", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY")

	// Read config file (ignore error if not found)
	_ = viper.ReadInConfig()
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// logError prints an error message to stderr.
func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// logInfo prints an info message to stderr (unless quiet mode).
func logInfo(format string, args ...any) {
	if !viper.GetBool("quiet") {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
