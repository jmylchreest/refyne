package refyne

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/jmylchreest/refyne/internal/crawler"
	"github.com/jmylchreest/refyne/internal/logger"
	"github.com/jmylchreest/refyne/pkg/cleaner"
	refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
	"github.com/jmylchreest/refyne/pkg/extractor"
	"github.com/jmylchreest/refyne/pkg/extractor/anthropic"
	"github.com/jmylchreest/refyne/pkg/extractor/ollama"
	"github.com/jmylchreest/refyne/pkg/extractor/openai"
	"github.com/jmylchreest/refyne/pkg/extractor/openrouter"
	"github.com/jmylchreest/refyne/pkg/fetcher"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// Exported error types for content validation.
// These are re-exported from internal/crawler for use by consumers.
var (
	// ErrInsufficientContent is returned when cleaned content is too small for extraction.
	// This typically indicates the page requires JavaScript rendering (dynamic fetch mode).
	ErrInsufficientContent = crawler.ErrInsufficientContent
)

// InsufficientContentError provides details about why content was insufficient.
// Use errors.As to check for this error type.
type InsufficientContentError = crawler.InsufficientContentError

// Version returns the module version of the refyne library.
// This returns the actual version consumers pulled via go get (e.g., "v1.0.0").
// Returns "(devel)" when built from source without version info.
func Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return "(unknown)"
}

// Result represents an extraction result.
type Result struct {
	URL             string
	FetchedAt       time.Time
	Data            any
	Raw             string // Raw LLM response
	RawContent      string // Raw page content sent to LLM (for training data)
	Errors          []schema.ValidationError
	TokenUsage      TokenUsage
	Model           string        // Actual model used (may differ from requested for auto-routing)
	Provider        string        // Provider name (anthropic, openai, openrouter, ollama)
	GenerationID    string        // Provider's generation ID (for cost tracking, e.g., OpenRouter)
	Cost            float64       // Actual cost in USD if provider returns it inline (check CostIncluded)
	CostIncluded    bool          // True if Cost contains actual cost from provider
	RetryCount      int
	FetchDuration   time.Duration // Time to fetch the page
	ExtractDuration time.Duration // Time for LLM extraction
	Error           error
}

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	InputTokens  int
	OutputTokens int
}

// Refyne is the main entry point for web scraping with LLM extraction.
type Refyne struct {
	fetcher   fetcher.Fetcher
	cleaner   cleaner.Cleaner
	extractor extractor.Extractor
	config    Config
}

// New creates a new Refyne instance.
func New(opts ...Option) (*Refyne, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Use injected fetcher or create a default static one
	var f fetcher.Fetcher
	if cfg.Fetcher != nil {
		f = cfg.Fetcher
	} else {
		f = fetcher.NewStatic(fetcher.StaticConfig{
			UserAgent: cfg.UserAgent,
			Timeout:   cfg.Timeout,
		})
	}

	// Use injected cleaner or create a default refyne cleaner with markdown output
	var cl cleaner.Cleaner
	if cfg.Cleaner != nil {
		cl = cfg.Cleaner
	} else {
		// Default: refyne cleaner with markdown output for LLM-optimized content
		refyneCfg := refynecleaner.DefaultConfig()
		refyneCfg.Output = refynecleaner.OutputMarkdown
		cl = refynecleaner.New(refyneCfg)
	}

	// Use injected extractor or create one based on provider
	var ext extractor.Extractor
	if cfg.Extractor != nil {
		ext = cfg.Extractor
	} else {
		llmCfg := &extractor.LLMConfig{
			Model:          cfg.Model,
			APIKey:         cfg.APIKey,
			BaseURL:        cfg.BaseURL,
			Temperature:    cfg.Temperature,
			MaxTokens:      cfg.MaxTokens,
			MaxRetries:     cfg.MaxRetries,
			MaxContentSize: cfg.MaxContentSize,
			StrictMode:     cfg.StrictMode,
		}

		var err error
		switch cfg.Provider {
		case "openai":
			ext, err = openai.New(llmCfg)
		case "openrouter":
			ext, err = openrouter.New(llmCfg)
		case "ollama":
			ext, err = ollama.New(llmCfg)
		case "anthropic", "":
			// Default to Anthropic
			ext, err = anthropic.New(llmCfg)
		default:
			return nil, fmt.Errorf("unknown provider: %s (use anthropic, openai, openrouter, or ollama)", cfg.Provider)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create extractor: %w", err)
		}
	}

	return &Refyne{
		fetcher:   f,
		cleaner:   cl,
		extractor: ext,
		config:    cfg,
	}, nil
}

// Extract fetches a single URL and extracts structured data.
func (r *Refyne) Extract(ctx context.Context, url string, s schema.Schema) (*Result, error) {
	// Prepare fetch options
	fetchOpts := fetcher.Options{
		UserAgent: r.config.UserAgent,
		Timeout:   r.config.Timeout,
	}

	// Fetch the page
	fetchStart := time.Now()
	content, err := r.fetcher.Fetch(ctx, url, fetchOpts)
	fetchDuration := time.Since(fetchStart)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Clean the content (convert HTML to markdown or other format)
	cleanStart := time.Now()
	cleanedContent, err := r.cleaner.Clean(content.HTML)
	cleanDuration := time.Since(cleanStart)
	if err != nil {
		// Fall back to fetcher's text extraction if cleaner fails
		logger.Debug("cleaner failed, using raw text",
			"cleaner", r.cleaner.Name(),
			"error", err)
		cleanedContent = content.Text
	} else {
		logger.Debug("content cleaned",
			"cleaner", r.cleaner.Name(),
			"input_size", len(content.HTML),
			"output_size", len(cleanedContent),
			"duration", cleanDuration)
	}

	// Extract data using cleaned content
	result, err := r.extractor.Extract(ctx, cleanedContent, s)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	return &Result{
		URL:        url,
		FetchedAt:  content.FetchedAt,
		Data:       result.Data,
		Raw:        result.Raw,
		RawContent: result.RawContent,
		TokenUsage: TokenUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		},
		Model:           result.Model,
		Provider:        result.Provider,
		GenerationID:    result.GenerationID,
		Cost:            result.Cost,
		CostIncluded:    result.CostIncluded,
		RetryCount:      result.RetryCount,
		FetchDuration:   fetchDuration,
		ExtractDuration: result.Duration,
		Errors:          result.Errors,
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

	// Create crawler with cleaner
	c := crawler.New(r.fetcher, r.cleaner, r.extractor, crawlCfg)

	// Start crawling
	crawlResults := c.Crawl(ctx, seeds, s)

	// Convert crawler results to public Result type
	results := make(chan *Result, 100)
	go func() {
		defer close(results)
		for cr := range crawlResults {
			result := &Result{
				URL:             cr.URL,
				FetchedAt:       cr.FetchedAt,
				Data:            cr.Data,
				Raw:             cr.Raw,
				FetchDuration:   cr.FetchDuration,
				ExtractDuration: cr.ExtractDuration,
				Errors:          cr.Errors,
				Error:           cr.Error,
			}
			// Copy extraction metadata if available (nil when extraction failed/skipped)
			if cr.Usage != nil {
				result.RawContent = cr.Usage.RawContent
				result.TokenUsage = TokenUsage{
					InputTokens:  cr.Usage.Usage.InputTokens,
					OutputTokens: cr.Usage.Usage.OutputTokens,
				}
				result.Model = cr.Usage.Model
				result.Provider = cr.Usage.Provider
				result.GenerationID = cr.Usage.GenerationID
				result.Cost = cr.Usage.Cost
				result.CostIncluded = cr.Usage.CostIncluded
				result.RetryCount = cr.Usage.RetryCount
			}
			results <- result
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

// Provider returns the extractor/provider name.
func (r *Refyne) Provider() string {
	return r.extractor.Name()
}
