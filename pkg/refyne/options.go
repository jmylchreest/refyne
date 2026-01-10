// Package refyne provides the public API for web scraping with LLM extraction.
package refyne

import (
	"time"

	"github.com/refyne/refyne/internal/crawler"
	"github.com/refyne/refyne/internal/scraper"
)

// Config holds all Refyne configuration.
type Config struct {
	// LLM settings
	Provider string
	Model    string
	APIKey   string
	BaseURL  string

	// Scraping settings
	FetchMode scraper.FetchMode
	UserAgent string
	Timeout   time.Duration

	// Extraction settings
	MaxRetries  int
	Temperature float64

	// Crawling settings
	CrawlConfig crawler.Config
}

// Chrome user agent for better compatibility with bot-protected sites
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Provider:    "anthropic",
		FetchMode:   scraper.FetchModeAuto,
		UserAgent:   defaultUserAgent,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		Temperature: 0.1,
		CrawlConfig: crawler.DefaultConfig(),
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

// WithFetchMode sets the fetch mode (auto, static, dynamic).
func WithFetchMode(mode scraper.FetchMode) Option {
	return func(c *Config) {
		c.FetchMode = mode
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
