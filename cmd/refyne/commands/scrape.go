package commands

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/internal/logger"
	"github.com/refyne/refyne/internal/output"
	"github.com/refyne/refyne/internal/scraper"
	"github.com/refyne/refyne/pkg/refyne"
	"github.com/refyne/refyne/pkg/schema"
)

// wrappedResult wraps extracted data with metadata.
type wrappedResult struct {
	Metadata resultMetadata `json:"_metadata"`
	Data     any            `json:"data"`
}

type resultMetadata struct {
	URL             string  `json:"url"`
	FetchedAt       string  `json:"fetched_at"`
	Model           string  `json:"model"`
	Provider        string  `json:"provider"`
	InputTokens     int     `json:"input_tokens"`
	OutputTokens    int     `json:"output_tokens"`
	FetchDurationMs int64   `json:"fetch_duration_ms"`
	LLMDurationMs   int64   `json:"llm_duration_ms"`
	RetryCount      int     `json:"retry_count,omitempty"`
}

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
	flags.StringP("provider", "p", "", "LLM provider: anthropic, openai, openrouter, ollama (auto-detects from env vars)")
	flags.StringP("model", "m", "", "model name (provider-specific)")
	flags.StringP("api-key", "k", "", "API key (or use env var)")
	flags.String("base-url", "", "custom API base URL")

	// Output settings
	flags.StringP("output", "o", "", "output file (default: stdout)")
	flags.String("format", "json", "output format: json, jsonl, yaml")
	flags.Bool("include-metadata", false, "wrap output with _metadata (url, fetched_at) and data keys")

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
	flags.Int("max-urls", 0, "max total URLs to process (0=unlimited)")
	flags.Duration("delay", 200*time.Millisecond, "delay between requests")
	flags.IntP("concurrency", "c", 3, "concurrent requests")

	// Required flags
	_ = scrapeCmd.MarkFlagRequired("schema")

	// Bind to viper
	_ = viper.BindPFlag("provider", flags.Lookup("provider"))
	_ = viper.BindPFlag("model", flags.Lookup("model"))
	_ = viper.BindPFlag("api_key", flags.Lookup("api-key"))
	_ = viper.BindPFlag("base_url", flags.Lookup("base-url"))
}

func runScrape(cmd *cobra.Command, args []string) error {
	// Initialize logger based on flags
	logger.Init(logger.Options{
		Debug: viper.GetBool("debug"),
		Quiet: viper.GetBool("quiet"),
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Debug("scrape command starting")

	// Get URLs
	urls, _ := cmd.Flags().GetStringSlice("url")
	if len(urls) == 0 {
		return cmd.Help()
	}
	logger.Debug("URLs to process", "count", len(urls), "urls", urls)

	// Load schema
	schemaPath, _ := cmd.Flags().GetString("schema")
	logger.Debug("loading schema", "path", schemaPath)
	s, err := schema.FromFile(schemaPath)
	if err != nil {
		logger.Error("failed to load schema", "error", err)
		return err
	}
	logger.Debug("schema loaded", "name", s.Name, "fields", len(s.Fields))

	// Get fetch mode
	fetchModeStr, _ := cmd.Flags().GetString("fetch-mode")
	fetchMode := scraper.FetchMode(fetchModeStr)
	logger.Debug("fetch mode", "mode", fetchModeStr)

	// Get timeout
	timeout, _ := cmd.Flags().GetDuration("timeout")
	logger.Debug("timeout", "duration", timeout)

	// Get max retries
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	logger.Debug("max retries", "retries", maxRetries)

	// Determine provider and model
	// Priority: explicit flags > viper config > auto-detection
	provider := viper.GetString("provider")
	model := viper.GetString("model")
	apiKey := viper.GetString("api_key")

	// Auto-detect provider if not explicitly set
	if provider == "" {
		detectedProvider, detectedKey := llm.DetectProvider()
		provider = detectedProvider
		if apiKey == "" {
			apiKey = detectedKey
		}
		logger.Debug("auto-detected provider", "provider", provider)
	}

	// Use default model for provider if not explicitly set
	if model == "" {
		model = llm.GetDefaultModel(provider)
		logger.Debug("using default model", "model", model)
	}

	logger.Debug("creating refyne instance",
		"provider", provider,
		"model", model,
		"has_api_key", apiKey != "")

	r, err := refyne.New(
		refyne.WithProvider(provider),
		refyne.WithModel(model),
		refyne.WithAPIKey(apiKey),
		refyne.WithBaseURL(viper.GetString("base_url")),
		refyne.WithFetchMode(fetchMode),
		refyne.WithTimeout(timeout),
		refyne.WithMaxRetries(maxRetries),
	)
	if err != nil {
		logger.Error("failed to initialize", "error", err)
		return err
	}
	defer func() { _ = r.Close() }()
	logger.Debug("refyne instance created")

	// Setup output
	outFile := os.Stdout
	if outPath, _ := cmd.Flags().GetString("output"); outPath != "" {
		f, err := os.Create(outPath) //#nosec G304 -- CLI tool writes to user-specified output file
		if err != nil {
			logger.Error("failed to create output file", "path", outPath, "error", err)
			return err
		}
		defer func() { _ = f.Close() }()
		outFile = f
	}

	formatStr, _ := cmd.Flags().GetString("format")
	writer, err := output.NewWriter(outFile, output.Format(formatStr))
	if err != nil {
		logger.Error("failed to create output writer", "format", formatStr, "error", err)
		return err
	}
	defer func() { _ = writer.Close() }()

	// Get metadata option
	includeMetadata, _ := cmd.Flags().GetBool("include-metadata")

	// Get crawling options
	followSelector, _ := cmd.Flags().GetString("follow")
	followPattern, _ := cmd.Flags().GetString("follow-pattern")
	nextSelector, _ := cmd.Flags().GetString("next")
	maxDepth, _ := cmd.Flags().GetInt("max-depth")
	maxPages, _ := cmd.Flags().GetInt("max-pages")
	maxURLs, _ := cmd.Flags().GetInt("max-urls")
	delay, _ := cmd.Flags().GetDuration("delay")
	concurrency, _ := cmd.Flags().GetInt("concurrency")

	// Determine if we're doing simple extraction or crawling
	isCrawling := followSelector != "" || followPattern != "" || nextSelector != ""

	var hasErrors bool

	if isCrawling {
		// Crawling mode
		logger.Info("starting crawl",
			"seeds", len(urls),
			"provider", provider,
			"model", model,
			"concurrency", concurrency,
			"delay", delay)

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
		if maxURLs > 0 {
			crawlOpts = append(crawlOpts, refyne.WithMaxURLs(maxURLs))
		}

		results := r.CrawlMany(ctx, urls, s, crawlOpts...)

		count := 0
		errorCount := 0
		for result := range results {
			if result.Error != nil {
				errorCount++
				hasErrors = true
				continue
			}

			if result.Data != nil {
				out := any(result.Data)
				if includeMetadata {
					out = wrappedResult{
						Metadata: resultMetadata{
							URL:             result.URL,
							FetchedAt:       result.FetchedAt.Format(time.RFC3339),
							Model:           result.Model,
							Provider:        result.Provider,
							InputTokens:     result.TokenUsage.InputTokens,
							OutputTokens:    result.TokenUsage.OutputTokens,
							FetchDurationMs: result.FetchDuration.Milliseconds(),
							LLMDurationMs:   result.ExtractDuration.Milliseconds(),
							RetryCount:      result.RetryCount,
						},
						Data: result.Data,
					}
				}
				if err := writer.Write(out); err != nil {
					logger.Error("failed to write output", "error", err)
					return err
				}
				count++
			}
		}

		logger.Info("crawl complete", "extracted", count, "errors", errorCount)
	} else {
		// Simple extraction mode
		logger.Info("starting extraction",
			"urls", len(urls),
			"provider", provider,
			"model", model,
			"concurrency", concurrency)

		results := r.ExtractMany(ctx, urls, s, concurrency)

		count := 0
		errorCount := 0
		for result := range results {
			if result.Error != nil {
				errorCount++
				hasErrors = true
				continue
			}

			out := any(result.Data)
			if includeMetadata {
				out = wrappedResult{
					Metadata: resultMetadata{
						URL:             result.URL,
						FetchedAt:       result.FetchedAt.Format(time.RFC3339),
						Model:           result.Model,
						Provider:        result.Provider,
						InputTokens:     result.TokenUsage.InputTokens,
						OutputTokens:    result.TokenUsage.OutputTokens,
						FetchDurationMs: result.FetchDuration.Milliseconds(),
						LLMDurationMs:   result.ExtractDuration.Milliseconds(),
						RetryCount:      result.RetryCount,
					},
					Data: result.Data,
				}
			}
			if err := writer.Write(out); err != nil {
				logger.Error("failed to write output", "error", err)
				return err
			}
			count++
		}

		logger.Info("extraction complete", "extracted", count, "errors", errorCount)
	}

	if hasErrors {
		return nil // Don't return error, we already logged individual errors
	}

	return nil
}
