package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	clifetcher "github.com/jmylchreest/refyne/cmd/refyne/fetcher"
	"github.com/jmylchreest/refyne/internal/logger"
	"github.com/jmylchreest/refyne/internal/output"
	"github.com/jmylchreest/refyne/pkg/cleaner"
	refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
	"github.com/jmylchreest/refyne/pkg/extractor"
	"github.com/jmylchreest/refyne/pkg/extractor/anthropic"
	"github.com/jmylchreest/refyne/pkg/extractor/ollama"
	"github.com/jmylchreest/refyne/pkg/extractor/openai"
	"github.com/jmylchreest/refyne/pkg/extractor/openrouter"
	"github.com/jmylchreest/refyne/pkg/fetcher"
	"github.com/jmylchreest/refyne/pkg/refyne"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// wrappedResult wraps extracted data with metadata.
type wrappedResult struct {
	Metadata resultMetadata `json:"_metadata"`
	Data     any            `json:"data"`
}

type resultMetadata struct {
	URL             string `json:"url"`
	FetchedAt       string `json:"fetched_at"`
	Model           string `json:"model"`
	Provider        string `json:"provider"`
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	FetchDurationMs int64  `json:"fetch_duration_ms"`
	LLMDurationMs   int64  `json:"llm_duration_ms"`
	RetryCount      int    `json:"retry_count,omitempty"`
}

// trainingDataRecord is a single input/output pair for fine-tuning.
type trainingDataRecord struct {
	URL    string `json:"url"`
	Input  string `json:"input"`  // Raw page content
	Output any    `json:"output"` // Extracted JSON data
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
	flags.Bool("include-metadata", true, "wrap output with _metadata and data keys (use --include-metadata=false to disable)")
	flags.String("save-training-data", "", "save input/output pairs for fine-tuning to this file (JSONL)")

	// Fetch settings
	flags.String("fetch-mode", "static", "fetch mode: static, dynamic")
	flags.Duration("timeout", 30*time.Second, "request timeout")
	flags.Bool("stealth", false, "enable anti-bot detection evasion for dynamic fetch mode")
	flags.Bool("googlebot", false, "spoof Googlebot user-agent (sites often whitelist Googlebot)")
	flags.String("flaresolverr-url", "", "FlareSolverr API URL for Cloudflare bypass (e.g., http://localhost:8191/v1)")

	// Extraction settings
	flags.Int("max-retries", 3, "max extraction retries")
	flags.String("max-content-size", "100KB", "max input content size (e.g., 100KB, 1MB, 0=unlimited)")
	flags.Bool("no-cleanse", false, "disable content cleaning (pass raw HTML to LLM)")

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
	logger.Debug("fetch mode", "mode", fetchModeStr)

	// Get timeout
	timeout, _ := cmd.Flags().GetDuration("timeout")
	logger.Debug("timeout", "duration", timeout)

	// Get max retries
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	logger.Debug("max retries", "retries", maxRetries)

	// Get max content size (0 or empty means unlimited)
	maxContentSizeStr, _ := cmd.Flags().GetString("max-content-size")
	var maxContentSize int
	if strings.TrimSpace(maxContentSizeStr) != "" && maxContentSizeStr != "0" {
		bytes, err := humanize.ParseBytes(maxContentSizeStr)
		if err != nil {
			logger.Error("invalid max-content-size", "value", maxContentSizeStr, "error", err)
			return err
		}
		maxContentSize = int(bytes)
	}
	logger.Debug("max content size", "bytes", maxContentSize)

	// Get fetcher options
	stealth, _ := cmd.Flags().GetBool("stealth")
	googlebot, _ := cmd.Flags().GetBool("googlebot")
	flareSolverrURL, _ := cmd.Flags().GetString("flaresolverr-url")

	// Create fetcher based on mode
	var f fetcher.Fetcher
	switch fetchModeStr {
	case "dynamic":
		// Use CLI's dynamic fetcher with advanced options
		dynamicFetcher, err := clifetcher.NewDynamicFetcher(clifetcher.Config{
			Timeout:         timeout,
			Stealth:         stealth,
			Googlebot:       googlebot,
			FlareSolverrURL: flareSolverrURL,
		})
		if err != nil {
			logger.Error("failed to create dynamic fetcher", "error", err)
			return err
		}
		f = dynamicFetcher
	case "static", "":
		// Use static fetcher (default)
		f = fetcher.NewStatic(fetcher.StaticConfig{
			Timeout: timeout,
		})
	default:
		return fmt.Errorf("unknown fetch mode: %s (use 'static' or 'dynamic')", fetchModeStr)
	}
	// Note: fetcher is closed by refyne.Close()

	// Create cleaner based on --no-cleanse flag
	noCleanse, _ := cmd.Flags().GetBool("no-cleanse")
	var cl cleaner.Cleaner
	if noCleanse {
		cl = cleaner.NewNoop()
		logger.Debug("content cleaning disabled")
	} else {
		// Default: Refyne cleaner with LLM-optimized markdown output
		// Images are extracted to frontmatter with {{IMG_001}} placeholders in body
		cfg := refynecleaner.DefaultConfig()
		cfg.Output = refynecleaner.OutputMarkdown
		cfg.IncludeFrontmatter = true
		cfg.ExtractImages = true
		cfg.ExtractHeadings = true
		cl = refynecleaner.New(cfg)
		logger.Debug("using refyne cleaner with markdown output", "cleaner", cl.Name())
	}

	// Build extractor fallback chain
	// Order: --provider flag first (if set), then config fallback_order, default: openrouter → anthropic → ollama
	preferredProvider := viper.GetString("provider")
	modelOverride := viper.GetString("model") // Override model for preferred provider

	llmCfg := &extractor.LLMConfig{
		MaxRetries:     maxRetries,
		MaxContentSize: maxContentSize,
	}

	ext, err := buildExtractorChain(preferredProvider, modelOverride, llmCfg)
	if err != nil {
		logger.Error("failed to build extractor chain", "error", err)
		return err
	}

	if !ext.Available() {
		logger.Error("no extractors available - set an API key or run Ollama locally")
		return fmt.Errorf("no extractors available")
	}

	logger.Debug("extractor chain built", "chain", ext.Name())

	r, err := refyne.New(
		refyne.WithFetcher(f),
		refyne.WithCleaner(cl),
		refyne.WithExtractor(ext),
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

	// Setup training data output if requested
	trainingDataPath, _ := cmd.Flags().GetString("save-training-data")
	var trainingFile *os.File
	var trainingEncoder *json.Encoder
	if trainingDataPath != "" {
		f, err := os.Create(trainingDataPath) //#nosec G304 -- CLI tool writes to user-specified file
		if err != nil {
			logger.Error("failed to create training data file", "path", trainingDataPath, "error", err)
			return err
		}
		defer func() { _ = f.Close() }()
		trainingFile = f
		trainingEncoder = json.NewEncoder(trainingFile)
		logger.Info("saving training data", "path", trainingDataPath)
	}

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
			"extractors", ext.Name(),
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
				// Write training data if requested
				if trainingEncoder != nil && result.RawContent != "" {
					record := trainingDataRecord{
						URL:    result.URL,
						Input:  result.RawContent,
						Output: result.Data,
					}
					if err := trainingEncoder.Encode(record); err != nil {
						logger.Error("failed to write training data", "error", err)
					}
				}
				count++
			}
		}

		logger.Info("crawl complete", "extracted", count, "errors", errorCount)
	} else {
		// Simple extraction mode
		logger.Info("starting extraction",
			"urls", len(urls),
			"extractors", ext.Name(),
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
			// Write training data if requested
			if trainingEncoder != nil && result.RawContent != "" {
				record := trainingDataRecord{
					URL:    result.URL,
					Input:  result.RawContent,
					Output: result.Data,
				}
				if err := trainingEncoder.Encode(record); err != nil {
					logger.Error("failed to write training data", "error", err)
				}
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

// ProviderConfig holds provider-specific settings from config file.
type ProviderConfig struct {
	Model       string  `mapstructure:"model"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	BaseURL     string  `mapstructure:"base_url"`
}

// Default fallback order: openrouter → anthropic → ollama
var defaultFallbackOrder = []string{"openrouter", "anthropic", "ollama"}

// buildExtractorChain creates a fallback extractor chain based on config.
// If preferredProvider is set (via --provider flag), it goes first.
// If modelOverride is set (via --model flag), it overrides the preferred provider's model.
// Then uses fallback_order from config, or default: openrouter → anthropic → ollama.
// Only adds providers that have API keys (except ollama which is always available).
func buildExtractorChain(preferredProvider, modelOverride string, baseCfg *extractor.LLMConfig) (*extractor.FallbackExtractor, error) {
	var extractors []extractor.Extractor
	added := make(map[string]bool)

	// Get fallback order from config, or use default
	fallbackOrder := viper.GetStringSlice("fallback_order")
	if len(fallbackOrder) == 0 {
		fallbackOrder = defaultFallbackOrder
	}

	// If preferred provider specified, put it first
	if preferredProvider != "" {
		// Remove from fallback order if present, then prepend
		var newOrder []string
		newOrder = append(newOrder, preferredProvider)
		for _, p := range fallbackOrder {
			if p != preferredProvider {
				newOrder = append(newOrder, p)
			}
		}
		fallbackOrder = newOrder
	}

	// Get provider-specific configs
	providerConfigs := make(map[string]ProviderConfig)
	_ = viper.UnmarshalKey("providers", &providerConfigs)

	// Helper to get provider config merged with base config
	getProviderConfig := func(name string, isPreferred bool) *extractor.LLMConfig {
		cfg := &extractor.LLMConfig{
			MaxRetries:     baseCfg.MaxRetries,
			MaxContentSize: baseCfg.MaxContentSize,
		}

		// Apply provider-specific config from file
		if pc, ok := providerConfigs[name]; ok {
			if pc.Model != "" {
				cfg.Model = pc.Model
			}
			if pc.Temperature > 0 {
				cfg.Temperature = pc.Temperature
			}
			if pc.MaxTokens > 0 {
				cfg.MaxTokens = pc.MaxTokens
			}
			if pc.BaseURL != "" {
				cfg.BaseURL = pc.BaseURL
			}
		}

		// If this is the preferred provider and model override is set, use it
		if isPreferred && modelOverride != "" {
			cfg.Model = modelOverride
		}

		return cfg
	}

	// Helper to add an extractor if not already added
	addExtractor := func(name string, ext extractor.Extractor, err error) bool {
		if err != nil {
			logger.Debug("failed to create extractor", "provider", name, "error", err)
			return false
		}
		if added[name] {
			return false
		}
		added[name] = true
		extractors = append(extractors, ext)
		logger.Debug("added extractor to chain", "provider", name, "available", ext.Available())
		return true
	}

	// Build chain in fallback order
	for _, provider := range fallbackOrder {
		cfg := getProviderConfig(provider, provider == preferredProvider)

		switch provider {
		case "anthropic":
			apiKey := os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				continue // Skip if no API key
			}
			cfg.APIKey = apiKey
			ext, err := anthropic.New(cfg)
			addExtractor("anthropic", ext, err)

		case "openai":
			apiKey := os.Getenv("OPENAI_API_KEY")
			if apiKey == "" {
				continue
			}
			cfg.APIKey = apiKey
			ext, err := openai.New(cfg)
			addExtractor("openai", ext, err)

		case "openrouter":
			apiKey := os.Getenv("OPENROUTER_API_KEY")
			if apiKey == "" {
				continue
			}
			cfg.APIKey = apiKey
			ext, err := openrouter.New(cfg)
			addExtractor("openrouter", ext, err)

		case "ollama":
			// Ollama doesn't need an API key - always add it
			ext, err := ollama.New(cfg)
			addExtractor("ollama", ext, err)

		default:
			logger.Debug("unknown provider in fallback_order", "provider", provider)
		}
	}

	if len(extractors) == 0 {
		return nil, fmt.Errorf("no extractors could be created")
	}

	return extractor.NewFallback(extractors...), nil
}
