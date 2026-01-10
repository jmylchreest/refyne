package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/refyne/refyne/internal/extractor"
	"github.com/refyne/refyne/internal/logger"
	"github.com/refyne/refyne/internal/scraper"
	"github.com/refyne/refyne/pkg/schema"
)

// Result represents a single crawl/extraction result.
type Result struct {
	URL             string
	Data            any
	Raw             string
	Errors          []schema.ValidationError
	Usage           extractor.ExtractionResult
	Error           error
	Depth           int
	FetchedAt       time.Time
	FetchDuration   time.Duration
	ExtractDuration time.Duration
}

// Config holds crawler configuration.
type Config struct {
	// Link following
	FollowSelector string // CSS selector for links to follow
	FollowPattern  string // Regex pattern for URLs to follow
	SameDomainOnly bool   // Only follow links on same domain (default: true)
	MaxDepth       int    // Max link depth (0 = seed only, 1 = seed + direct links)

	// Pagination
	NextSelector string // CSS selector for "next page" link
	MaxPages     int    // Max pages to crawl (0 = unlimited)

	// Limits
	MaxURLs int // Max total URLs to process (0 = unlimited)

	// Rate limiting
	Delay       time.Duration // Delay between requests
	Concurrency int           // Max concurrent requests

	// Extraction
	ExtractFromSeeds bool // Whether to extract from seed pages (vs just follow links)
}

// DefaultConfig returns sensible crawler defaults.
func DefaultConfig() Config {
	return Config{
		SameDomainOnly:   true,
		MaxDepth:         1,
		MaxPages:         0, // unlimited
		MaxURLs:          0, // unlimited
		Delay:            200 * time.Millisecond,
		Concurrency:      3,
		ExtractFromSeeds: false,
	}
}

// Crawler orchestrates multi-page crawling and extraction.
type Crawler struct {
	fetcher   scraper.Fetcher
	extractor *extractor.Extractor
	config    Config
}

// New creates a new Crawler.
func New(fetcher scraper.Fetcher, ext *extractor.Extractor, cfg Config) *Crawler {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	return &Crawler{
		fetcher:   fetcher,
		extractor: ext,
		config:    cfg,
	}
}

// Crawl starts crawling from seed URLs and returns results via channel.
func (c *Crawler) Crawl(ctx context.Context, seeds []string, s schema.Schema) <-chan Result {
	results := make(chan Result, 100)

	go func() {
		defer close(results)
		c.crawl(ctx, seeds, s, results)
	}()

	return results
}

func (c *Crawler) crawl(ctx context.Context, seeds []string, s schema.Schema, results chan<- Result) {
	logger.Debug("crawler starting",
		"seeds", len(seeds),
		"max_depth", c.config.MaxDepth,
		"max_urls", c.config.MaxURLs,
		"concurrency", c.config.Concurrency,
		"delay", c.config.Delay)

	// Log seed URLs at Info level
	for _, seed := range seeds {
		logger.Info("seed", "url", seed)
	}

	queue := NewURLQueue()
	var linkSelector *LinkSelector
	var paginationSelector *PaginationSelector

	// Setup link selector if configured
	if c.config.FollowSelector != "" || c.config.FollowPattern != "" {
		logger.Debug("crawler setting up link selector",
			"css_selector", c.config.FollowSelector,
			"pattern", c.config.FollowPattern)
		var err error
		linkSelector, err = NewLinkSelector(c.config.FollowSelector, c.config.FollowPattern)
		if err != nil {
			logger.Debug("crawler invalid link selector", "error", err)
			results <- Result{Error: fmt.Errorf("invalid link selector: %w", err)}
			return
		}
	}

	// Setup pagination selector if configured
	if c.config.NextSelector != "" {
		logger.Debug("crawler setting up pagination selector", "selector", c.config.NextSelector)
		paginationSelector = NewPaginationSelector(c.config.NextSelector)
	}

	// Add seed URLs to queue at depth 0
	for _, seed := range seeds {
		logger.Debug("crawler adding seed URL", "url", seed)
		queue.Add(seed, 0)
	}

	// Track processed URLs
	urlsProcessed := 0
	paginationPages := 0

	// Semaphore for concurrency control
	sem := make(chan struct{}, c.config.Concurrency)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
		}

		// Check max URLs limit
		if c.config.MaxURLs > 0 && urlsProcessed >= c.config.MaxURLs {
			logger.Debug("crawler reached max URLs limit", "max_urls", c.config.MaxURLs)
			wg.Wait()
			return
		}

		// Get next URL
		currentURL, depth, ok := queue.Pop()
		if !ok {
			// Queue empty, wait for in-flight requests
			wg.Wait()
			// Check if queue is still empty
			if queue.Len() == 0 {
				return
			}
			continue
		}

		// Check max pages for pagination (depth 0 pages only)
		if depth == 0 && c.config.MaxPages > 0 && paginationPages >= c.config.MaxPages {
			logger.Debug("crawler reached max pagination pages", "max_pages", c.config.MaxPages)
			continue
		}

		// Acquire semaphore
		sem <- struct{}{}
		wg.Add(1)

		go func(url string, d int) {
			defer wg.Done()
			defer func() { <-sem }()

			// Rate limiting
			if c.config.Delay > 0 {
				time.Sleep(c.config.Delay)
			}

			c.processURL(ctx, url, d, s, queue, linkSelector, paginationSelector, results)
		}(currentURL, depth)

		urlsProcessed++
		if depth == 0 {
			paginationPages++
		}
	}
}

func (c *Crawler) processURL(
	ctx context.Context,
	url string,
	depth int,
	s schema.Schema,
	queue *URLQueue,
	linkSelector *LinkSelector,
	paginationSelector *PaginationSelector,
	results chan<- Result,
) {
	logger.Debug("crawler processing URL", "url", url, "depth", depth)

	// Fetch the page
	fetchStart := time.Now()
	content, err := c.fetcher.Fetch(ctx, url, scraper.DefaultFetchOptions())
	fetchDuration := time.Since(fetchStart)

	if err != nil {
		logger.Info("fetch failed", "url", url, "error", err, "duration", fetchDuration)
		results <- Result{URL: url, Depth: depth, Error: fmt.Errorf("fetch error: %w", err), FetchDuration: fetchDuration}
		return
	}
	logger.Debug("crawler fetch complete",
		"url", url,
		"text_size", len(content.Text),
		"links_count", len(content.Links))

	// Check if we should extract from this page
	shouldExtract := false
	if depth == 0 && c.config.ExtractFromSeeds {
		// Seed page with extraction enabled
		shouldExtract = true
	} else if depth > 0 {
		// Detail page (followed from seed)
		shouldExtract = true
	} else if linkSelector == nil {
		// No link following configured, extract from seed
		shouldExtract = true
	}

	// Extract data if appropriate
	var extractDuration time.Duration
	if shouldExtract {
		extractStart := time.Now()
		extractResult, err := c.extractor.Extract(ctx, content.Text, s)
		extractDuration = time.Since(extractStart)

		if err != nil {
			logger.Info("extraction failed",
				"url", url,
				"fetch", fetchDuration.Round(time.Millisecond),
				"extract", extractDuration.Round(time.Millisecond),
				"error", err)
			results <- Result{
				URL:             url,
				Depth:           depth,
				Error:           fmt.Errorf("extraction error: %w", err),
				FetchedAt:       content.FetchedAt,
				FetchDuration:   fetchDuration,
				ExtractDuration: extractDuration,
			}
		} else {
			logger.Debug("crawler extraction complete",
				"url", url,
				"validation_errors", len(extractResult.Errors))
			logger.Info("extracted",
				"url", url,
				"fetch", fetchDuration.Round(time.Millisecond),
				"llm", extractResult.LLMDuration.Round(time.Millisecond),
				"tokens", extractResult.Usage.InputTokens+extractResult.Usage.OutputTokens)
			results <- Result{
				URL:             url,
				Depth:           depth,
				Data:            extractResult.Data,
				Raw:             extractResult.Raw,
				Errors:          extractResult.Errors,
				Usage:           extractResult,
				FetchedAt:       content.FetchedAt,
				FetchDuration:   fetchDuration,
				ExtractDuration: extractDuration,
			}
		}
	} else {
		logger.Info("fetched (no extraction)", "url", url, "fetch", fetchDuration.Round(time.Millisecond), "links", len(content.Links))
	}

	// Follow links if configured and within depth limit
	if linkSelector != nil && depth < c.config.MaxDepth {
		links, err := linkSelector.ExtractLinks(content.HTML, url)
		if err == nil {
			logger.Debug("crawler found links to follow", "url", url, "links_count", len(links))
			addedCount := 0
			for _, link := range links {
				// Check same domain constraint
				if c.config.SameDomainOnly && !IsSameDomain(url, link) {
					logger.Debug("crawler skipping cross-domain link", "link", link)
					continue
				}
				if queue.Add(link, depth+1) {
					addedCount++
				}
			}
			if addedCount > 0 {
				logger.Info("following links", "from", url, "count", addedCount)
			}
		} else {
			logger.Debug("crawler link extraction failed", "url", url, "error", err)
		}
	}

	// Handle pagination (only at depth 0)
	if paginationSelector != nil && depth == 0 {
		if nextURL, found := paginationSelector.FindNextPage(content.HTML, url); found {
			logger.Debug("crawler found next page", "next_url", nextURL)
			logger.Info("pagination", "next", nextURL)
			// Pagination stays at depth 0
			queue.Add(nextURL, 0)
		}
	}

	logger.Debug("crawler finished processing URL", "url", url)
}
