// Package refyne provides the public API for web scraping with LLM extraction.
package refyne

import (
	"io"
	"log/slog"
	"time"

	"github.com/jmylchreest/refyne/internal/crawler"
	"github.com/jmylchreest/refyne/internal/logger"
	"github.com/jmylchreest/refyne/pkg/cleaner"
	"github.com/jmylchreest/refyne/pkg/extractor"
	"github.com/jmylchreest/refyne/pkg/fetcher"
)

// Config holds all Refyne configuration.
type Config struct {
	// LLM settings
	Provider string
	Model    string
	APIKey   string
	BaseURL  string

	// Scraping settings
	UserAgent string
	Timeout   time.Duration
	Fetcher   fetcher.Fetcher   // Optional: inject a pre-configured fetcher
	Cleaner   cleaner.Cleaner   // Optional: inject a content cleaner (default: markdown)
	Extractor extractor.Extractor // Optional: inject a custom extractor

	// Extraction settings (used when Extractor is nil)
	MaxRetries     int
	Temperature    float64
	MaxTokens      int  // Max output tokens for LLM responses (0 = default 8192)
	MaxContentSize int  // Max input content size in bytes (0 = default 100KB)
	StrictMode     bool // Use strict JSON schema mode (only supported by some models)

	// Crawling settings
	CrawlConfig crawler.Config
}

// Chrome user agent for better compatibility with bot-protected sites
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Provider:       "anthropic",
		UserAgent:      defaultUserAgent,
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		Temperature:    0.1,
		MaxContentSize: 100000, // 100KB default (SI units)
		CrawlConfig:    crawler.DefaultConfig(),
	}
}

// Option configures Refyne.
type Option func(*Config)

// WithProvider sets the LLM provider.
func WithProvider(provider string) Option {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithModel sets the LLM model.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(key string) Option {
	return func(c *Config) {
		c.APIKey = key
	}
}

// WithBaseURL sets a custom API base URL.
func WithBaseURL(url string) Option {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithUserAgent sets the HTTP user agent.
func WithUserAgent(ua string) Option {
	return func(c *Config) {
		c.UserAgent = ua
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

// WithMaxRetries sets the maximum extraction retry attempts.
func WithMaxRetries(n int) Option {
	return func(c *Config) {
		c.MaxRetries = n
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(t float64) Option {
	return func(c *Config) {
		c.Temperature = t
	}
}

// WithMaxTokens sets the maximum output tokens for LLM responses.
// Default is 8192 if not specified.
func WithMaxTokens(n int) Option {
	return func(c *Config) {
		c.MaxTokens = n
	}
}

// WithMaxContentSize sets the maximum input content size in bytes.
func WithMaxContentSize(n int) Option {
	return func(c *Config) {
		c.MaxContentSize = n
	}
}

// WithStrictMode enables or disables strict JSON schema mode.
// Only OpenAI and OpenAI-compatible models (gpt-4o, gpt-4o-mini) support strict mode.
// When enabled, the model must return valid JSON matching the schema exactly.
// When disabled (default), the model will try to follow the schema but may not be exact.
// Set to false for models like Gemini, Llama, Claude that don't support strict mode.
func WithStrictMode(strict bool) Option {
	return func(c *Config) {
		c.StrictMode = strict
	}
}

// WithFetcher injects a pre-configured fetcher.
// This allows the caller to configure dynamic fetching, stealth mode,
// Googlebot spoofing, FlareSolverr, and other fetcher-specific options.
// See cmd/refyne/fetcher for example implementations.
func WithFetcher(f fetcher.Fetcher) Option {
	return func(c *Config) {
		c.Fetcher = f
	}
}

// WithCleaner injects a content cleaner.
// The cleaner transforms fetched HTML into a format suitable for LLM extraction.
// Default: MarkdownCleaner (converts HTML to clean Markdown)
// Use cleaner.NewNoop() to pass content through unchanged.
func WithCleaner(cl cleaner.Cleaner) Option {
	return func(c *Config) {
		c.Cleaner = cl
	}
}

// WithExtractor injects a custom extractor.
// The extractor handles structured data extraction from cleaned content.
// Default: creates an LLM-based extractor using the configured provider.
// Use extractor.NewFallback() to try multiple extractors in order.
func WithExtractor(ext extractor.Extractor) Option {
	return func(c *Config) {
		c.Extractor = ext
	}
}

// WithLogger sets a custom slog.Logger for the refyne library.
// Note: Due to refyne's internal architecture, this sets a global logger
// that affects all refyne instances. For most use cases, call this once
// at startup when creating your first Refyne instance.
func WithLogger(l *slog.Logger) Option {
	return func(c *Config) {
		logger.SetLogger(l)
	}
}

// CrawlOption configures crawling behavior.
type CrawlOption func(*crawler.Config)

// WithFollowSelector sets the CSS selector for links to follow.
func WithFollowSelector(selector string) CrawlOption {
	return func(c *crawler.Config) {
		c.FollowSelector = selector
	}
}

// WithFollowPattern sets the regex pattern for URLs to follow.
func WithFollowPattern(pattern string) CrawlOption {
	return func(c *crawler.Config) {
		c.FollowPattern = pattern
	}
}

// WithMaxDepth sets the maximum link depth.
func WithMaxDepth(depth int) CrawlOption {
	return func(c *crawler.Config) {
		c.MaxDepth = depth
	}
}

// WithNextSelector sets the CSS selector for pagination.
func WithNextSelector(selector string) CrawlOption {
	return func(c *crawler.Config) {
		c.NextSelector = selector
	}
}

// WithMaxPages sets the maximum pagination pages to crawl.
func WithMaxPages(n int) CrawlOption {
	return func(c *crawler.Config) {
		c.MaxPages = n
	}
}

// WithMaxURLs sets the maximum total URLs to process.
func WithMaxURLs(n int) CrawlOption {
	return func(c *crawler.Config) {
		c.MaxURLs = n
	}
}

// WithDelay sets the delay between requests.
func WithDelay(d time.Duration) CrawlOption {
	return func(c *crawler.Config) {
		c.Delay = d
	}
}

// WithConcurrency sets the number of concurrent requests.
func WithConcurrency(n int) CrawlOption {
	return func(c *crawler.Config) {
		c.Concurrency = n
	}
}

// WithSameDomainOnly restricts crawling to the same domain.
func WithSameDomainOnly(enabled bool) CrawlOption {
	return func(c *crawler.Config) {
		c.SameDomainOnly = enabled
	}
}

// WithExtractFromSeeds enables extraction from seed pages.
func WithExtractFromSeeds(enabled bool) CrawlOption {
	return func(c *crawler.Config) {
		c.ExtractFromSeeds = enabled
	}
}

// WithOnURLsQueued sets a callback for when URLs are queued.
// The callback receives the total number of URLs queued (including already processed).
// Use this for progress tracking when the total URL count is not known upfront.
func WithOnURLsQueued(fn func(count int)) CrawlOption {
	return func(c *crawler.Config) {
		c.OnURLsQueued = fn
	}
}

// SetLogger sets a custom slog.Logger for the refyne library.
// This allows refyne logs to be integrated with your application's logging system.
// Call this once at application startup before creating any Refyne instances.
func SetLogger(l *slog.Logger) {
	logger.SetLogger(l)
}

// SetDebugLogging enables or disables debug-level logging for the refyne library.
// This is a global setting that affects all refyne instances.
// Call this once at application startup if you want verbose crawler logs.
func SetDebugLogging(enabled bool) {
	logger.Init(logger.Options{
		Debug: enabled,
	})
}

// SetLogOutput configures the refyne logger output destination and options.
// Use this to redirect refyne logs to a custom writer (e.g., to integrate with
// your application's logging system).
func SetLogOutput(output io.Writer, debug bool, jsonFormat bool) {
	logger.Init(logger.Options{
		Debug:  debug,
		JSON:   jsonFormat,
		Output: output,
	})
}
