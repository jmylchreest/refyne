package commands

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/refyne/refyne/internal/output"
	"github.com/refyne/refyne/internal/scraper"
	"github.com/refyne/refyne/pkg/refyne"
	"github.com/refyne/refyne/pkg/schema"
)

var scrapeCmd = &cobra.Command{
	Use:   "scrape",
	Short: "Extract structured data from URLs",
	Long: `Scrape web pages and extract structured data using LLM.

The schema file defines what data to extract. It can be JSON or YAML
and should include field names, types, and descriptions.

Examples:
  # Single page extraction
  refyne scrape -u "https://example.com/page" -s schema.json

  # Crawl with link following
  refyne scrape -u "https://example.com/list" -s schema.json \
      --follow "a.item" --max-depth 1

  # Pagination
  refyne scrape -u "https://example.com/search" -s schema.json \
      --follow "a.result" --next "a.next-page" --max-pages 5`,
	RunE: runScrape,
}

func init() {
	rootCmd.AddCommand(scrapeCmd)

	flags := scrapeCmd.Flags()

	// URL inputs
	flags.StringSliceP("url", "u", nil, "URL(s) to scrape (can be repeated)")
	flags.StringP("schema", "s", "", "path to schema file (required)")

	// LLM settings
	flags.StringP("provider", "p", "anthropic", "LLM provider: anthropic, openai, openrouter, ollama")
	flags.StringP("model", "m", "", "model name (provider-specific)")
	flags.StringP("api-key", "k", "", "API key (or use env var)")
	flags.String("base-url", "", "custom API base URL")

	// Output settings
	flags.StringP("output", "o", "", "output file (default: stdout)")
	flags.String("format", "json", "output format: json, jsonl, yaml")

	// Fetch settings
	flags.String("fetch-mode", "auto", "fetch mode: auto, static, dynamic")
	flags.Duration("timeout", 30*time.Second, "request timeout")

	// Extraction settings
	flags.Int("max-retries", 3, "max extraction retries")

	// Crawling settings
	flags.String("follow", "", "CSS selector for links to follow")
	flags.String("follow-pattern", "", "regex pattern for URLs to follow")
	flags.String("next", "", "CSS selector for pagination next link")
	flags.Int("max-depth", 1, "max link depth (0=seed only)")
	flags.Int("max-pages", 0, "max pagination pages (0=unlimited)")
	flags.Duration("delay", 500*time.Millisecond, "delay between requests")
	flags.IntP("concurrency", "c", 1, "concurrent requests")

	// Required flags
	scrapeCmd.MarkFlagRequired("schema")

	// Bind to viper
	viper.BindPFlag("provider", flags.Lookup("provider"))
	viper.BindPFlag("model", flags.Lookup("model"))
	viper.BindPFlag("api_key", flags.Lookup("api-key"))
	viper.BindPFlag("base_url", flags.Lookup("base-url"))
}

func runScrape(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Get URLs
	urls, _ := cmd.Flags().GetStringSlice("url")
	if len(urls) == 0 {
		return cmd.Help()
	}

	// Load schema
	schemaPath, _ := cmd.Flags().GetString("schema")
	s, err := schema.FromFile(schemaPath)
	if err != nil {
		logError("failed to load schema: %v", err)
		return err
	}

	// Get fetch mode
	fetchModeStr, _ := cmd.Flags().GetString("fetch-mode")
	fetchMode := scraper.FetchMode(fetchModeStr)

	// Get timeout
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Get max retries
	maxRetries, _ := cmd.Flags().GetInt("max-retries")

	// Create refyne instance
	r, err := refyne.New(
		refyne.WithProvider(viper.GetString("provider")),
		refyne.WithModel(viper.GetString("model")),
		refyne.WithAPIKey(viper.GetString("api_key")),
		refyne.WithBaseURL(viper.GetString("base_url")),
		refyne.WithFetchMode(fetchMode),
		refyne.WithTimeout(timeout),
		refyne.WithMaxRetries(maxRetries),
	)
	if err != nil {
		logError("failed to initialize: %v", err)
		return err
	}
	defer r.Close()

	// Setup output
	outFile := os.Stdout
	if outPath, _ := cmd.Flags().GetString("output"); outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			logError("failed to create output file: %v", err)
			return err
		}
		defer f.Close()
		outFile = f
	}

	formatStr, _ := cmd.Flags().GetString("format")
	writer, err := output.NewWriter(outFile, output.Format(formatStr))
	if err != nil {
		logError("failed to create output writer: %v", err)
		return err
	}
	defer writer.Close()

	// Get crawling options
	followSelector, _ := cmd.Flags().GetString("follow")
	followPattern, _ := cmd.Flags().GetString("follow-pattern")
	nextSelector, _ := cmd.Flags().GetString("next")
	maxDepth, _ := cmd.Flags().GetInt("max-depth")
	maxPages, _ := cmd.Flags().GetInt("max-pages")
	delay, _ := cmd.Flags().GetDuration("delay")
	concurrency, _ := cmd.Flags().GetInt("concurrency")

	// Determine if we're doing simple extraction or crawling
	isCrawling := followSelector != "" || followPattern != "" || nextSelector != ""

	var hasErrors bool

	if isCrawling {
		// Crawling mode
		logInfo("Starting crawl from %d seed URL(s)...", len(urls))

		crawlOpts := []refyne.CrawlOption{
			refyne.WithMaxDepth(maxDepth),
			refyne.WithDelay(delay),
			refyne.WithConcurrency(concurrency),
		}

		if followSelector != "" {
			crawlOpts = append(crawlOpts, refyne.WithFollowSelector(followSelector))
		}
		if followPattern != "" {
			crawlOpts = append(crawlOpts, refyne.WithFollowPattern(followPattern))
		}
		if nextSelector != "" {
			crawlOpts = append(crawlOpts, refyne.WithNextSelector(nextSelector))
		}
		if maxPages > 0 {
			crawlOpts = append(crawlOpts, refyne.WithMaxPages(maxPages))
		}

		results := r.CrawlMany(ctx, urls, s, crawlOpts...)

		count := 0
		for result := range results {
			if result.Error != nil {
				logError("Error processing %s: %v", result.URL, result.Error)
				hasErrors = true
				continue
			}

			if result.Data != nil {
				if err := writer.Write(result.Data); err != nil {
					logError("failed to write output: %v", err)
					return err
				}
				count++
				logInfo("Extracted: %s (tokens: %d)", result.URL, result.TokenUsage.InputTokens+result.TokenUsage.OutputTokens)
			}
		}

		logInfo("Completed: %d items extracted", count)
	} else {
		// Simple extraction mode
		logInfo("Extracting from %d URL(s)...", len(urls))

		results := r.ExtractMany(ctx, urls, s, concurrency)

		count := 0
		for result := range results {
			if result.Error != nil {
				logError("Error processing %s: %v", result.URL, result.Error)
				hasErrors = true
				continue
			}

			if err := writer.Write(result.Data); err != nil {
				logError("failed to write output: %v", err)
				return err
			}
			count++
			logInfo("Extracted: %s (tokens: %d)", result.URL, result.TokenUsage.InputTokens+result.TokenUsage.OutputTokens)
		}

		logInfo("Completed: %d items extracted", count)
	}

	if err := writer.Flush(); err != nil {
		logError("failed to flush output: %v", err)
		return err
	}

	if hasErrors {
		return nil // Don't return error, we already logged individual errors
	}

	return nil
}
