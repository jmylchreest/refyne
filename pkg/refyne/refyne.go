package refyne

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/refyne/refyne/internal/crawler"
	"github.com/refyne/refyne/internal/extractor"
	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/internal/scraper"
	"github.com/refyne/refyne/pkg/schema"
)

// Result represents an extraction result.
type Result struct {
	URL        string
	FetchedAt  time.Time
	Data       any
	Raw        string
	Errors     []schema.ValidationError
	TokenUsage TokenUsage
	RetryCount int
	Error      error
}

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// Refyne is the main entry point for web scraping with LLM extraction.
type Refyne struct {
	provider  llm.Provider
	fetcher   scraper.Fetcher
	extractor *extractor.Extractor
	config    Config
}

// New creates a new Refyne instance.
func New(opts ...Option) (*Refyne, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create LLM provider
	provider, err := llm.NewProvider(cfg.Provider, llm.ProviderConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Create fetcher
	fetcher, err := scraper.NewFetcher(cfg.FetchMode, scraper.FetcherConfig{
		UserAgent: cfg.UserAgent,
		Timeout:   cfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fetcher: %w", err)
	}

	// Create extractor
	ext := extractor.New(provider,
		extractor.WithMaxRetries(cfg.MaxRetries),
		extractor.WithTemperature(cfg.Temperature),
	)

	return &Refyne{
		provider:  provider,
		fetcher:   fetcher,
		extractor: ext,
		config:    cfg,
	}, nil
}

// Extract fetches a single URL and extracts structured data.
func (r *Refyne) Extract(ctx context.Context, url string, s schema.Schema) (*Result, error) {
	// Fetch the page
	content, err := r.fetcher.Fetch(ctx, url, scraper.FetchOptions{
		UserAgent: r.config.UserAgent,
		Timeout:   r.config.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Extract data
	result, err := r.extractor.Extract(ctx, content.Text, s)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return &Result{
		URL:       url,
		FetchedAt: content.FetchedAt,
		Data:      result.Data,
		Raw:       result.Raw,
		TokenUsage: TokenUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		},
		RetryCount: result.RetryCount,
		Errors:     result.Errors,
	}, nil
}

// ExtractMany extracts data from multiple URLs concurrently.
func (r *Refyne) ExtractMany(ctx context.Context, urls []string, s schema.Schema, concurrency int) <-chan *Result {
	if concurrency < 1 {
		concurrency = 1
	}

	results := make(chan *Result, len(urls))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := r.Extract(ctx, u, s)
			if err != nil {
				results <- &Result{URL: u, Error: err}
				return
			}
			results <- result
		}(url)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// Crawl starts a multi-page crawl from seed URLs.
func (r *Refyne) Crawl(ctx context.Context, seedURL string, s schema.Schema, opts ...CrawlOption) <-chan *Result {
	return r.CrawlMany(ctx, []string{seedURL}, s, opts...)
}

// CrawlMany starts a multi-page crawl from multiple seed URLs.
func (r *Refyne) CrawlMany(ctx context.Context, seeds []string, s schema.Schema, opts ...CrawlOption) <-chan *Result {
	// Apply crawl options
	crawlCfg := r.config.CrawlConfig
	for _, opt := range opts {
		opt(&crawlCfg)
	}

	// Create crawler
	c := crawler.New(r.fetcher, r.extractor, crawlCfg)

	// Start crawling
	crawlResults := c.Crawl(ctx, seeds, s)

	// Convert crawler results to public Result type
	results := make(chan *Result, 100)
	go func() {
		defer close(results)
		for cr := range crawlResults {
			results <- &Result{
				URL:       cr.URL,
				FetchedAt: cr.FetchedAt,
				Data:      cr.Data,
				Raw:       cr.Raw,
				TokenUsage: TokenUsage{
					InputTokens:  cr.Usage.Usage.InputTokens,
					OutputTokens: cr.Usage.Usage.OutputTokens,
				},
				RetryCount: cr.Usage.RetryCount,
				Errors:     cr.Errors,
				Error:      cr.Error,
			}
		}
	}()

	return results
}

// Close releases all resources.
func (r *Refyne) Close() error {
	if r.fetcher != nil {
		return r.fetcher.Close()
	}
	return nil
}

// Provider returns the underlying LLM provider name.
func (r *Refyne) Provider() string {
	return r.provider.Name()
}
