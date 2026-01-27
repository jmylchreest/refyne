package crawler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jmylchreest/refyne/internal/logger"
	"github.com/jmylchreest/refyne/pkg/cleaner"
	"github.com/jmylchreest/refyne/pkg/extractor"
	"github.com/jmylchreest/refyne/pkg/fetcher"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// ErrInsufficientContent is returned when the cleaned content is smaller than MinContentSize.
// This typically indicates the page requires JavaScript rendering (dynamic fetch mode).
var ErrInsufficientContent = errors.New("insufficient content for extraction")

// InsufficientContentError provides details about why content was insufficient.
type InsufficientContentError struct {
	ContentSize int // Actual content size in bytes
	MinRequired int // Minimum required size in bytes
}

func (e *InsufficientContentError) Error() string {
	return fmt.Sprintf("insufficient content: got %d bytes, need at least %d (page may require JavaScript rendering)", e.ContentSize, e.MinRequired)
}

func (e *InsufficientContentError) Unwrap() error {
	return ErrInsufficientContent
}

// Result represents a single crawl/extraction result.
type Result struct {
	URL             string
	Data            any
	Raw             string
	Errors          []schema.ValidationError
	Usage           *extractor.Result // Extraction result with token usage, model, etc.
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

	// Content validation
	MinContentSize int // Minimum cleaned content size in bytes (default: 200). Returns error if content is smaller.

	// Callbacks
	OnURLsQueued func(count int) // Called when URLs are queued (for progress tracking)
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
		MinContentSize:   200, // Minimum 200 bytes of cleaned content
	}
}

// Crawler orchestrates multi-page crawling and extraction.
type Crawler struct {
	fetcher   fetcher.Fetcher
	cleaner   cleaner.Cleaner
	extractor extractor.Extractor
	config    Config
}

// New creates a new Crawler.
func New(f fetcher.Fetcher, cl cleaner.Cleaner, ext extractor.Extractor, cfg Config) *Crawler {
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	return &Crawler{
		fetcher:   f,
		cleaner:   cl,
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

	// Notify about initial queued URLs
	if c.config.OnURLsQueued != nil {
		c.config.OnURLsQueued(queue.TotalQueued())
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
	content, err := c.fetcher.Fetch(ctx, url, fetcher.Options{})
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
		// Clean the HTML content before extraction
		cleanStart := time.Now()
		cleanedContent, err := c.cleaner.Clean(content.HTML)
		cleanDuration := time.Since(cleanStart)
		if err != nil {
			// Fall back to fetcher's text extraction if cleaner fails
			logger.Debug("cleaner failed, using raw text",
				"url", url,
				"cleaner", c.cleaner.Name(),
				"error", err)
			cleanedContent = content.Text
		} else {
			logger.Debug("content cleaned",
				"url", url,
				"cleaner", c.cleaner.Name(),
				"input_size", len(content.HTML),
				"output_size", len(cleanedContent),
				"duration", cleanDuration)
		}

		// Validate minimum content size before extraction
		// This prevents LLM hallucination on pages with insufficient content
		// (e.g., JavaScript-heavy sites that need browser rendering)
		minSize := c.config.MinContentSize
		if minSize > 0 && len(cleanedContent) < minSize {
			logger.Info("insufficient content for extraction",
				"url", url,
				"content_size", len(cleanedContent),
				"min_required", minSize,
				"hint", "page may require JavaScript rendering (dynamic fetch mode)")
			results <- Result{
				URL:           url,
				Depth:         depth,
				Error:         &InsufficientContentError{ContentSize: len(cleanedContent), MinRequired: minSize},
				FetchedAt:     content.FetchedAt,
				FetchDuration: fetchDuration,
			}
			return
		}

		extractStart := time.Now()
		extractResult, err := c.extractor.Extract(ctx, cleanedContent, s)
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
				"fetch", fetchDuration.Round(time.Millisecond),
				"llm", extractResult.Duration.Round(time.Millisecond),
				"input_tokens", extractResult.Usage.InputTokens,
				"output_tokens", extractResult.Usage.OutputTokens,
				"validation_errors", len(extractResult.Errors))
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
		logger.Debug("fetched (no extraction)", "url", url, "fetch", fetchDuration.Round(time.Millisecond), "links", len(content.Links))
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
					logger.Debug("crawler queued link", "link", link, "depth", depth+1)
					addedCount++
				} else {
					logger.Debug("crawler skipping already-seen link", "link", link)
				}
			}
			if addedCount > 0 {
				logger.Info("following links", "from", url, "count", addedCount)
				// Notify about newly queued URLs
				if c.config.OnURLsQueued != nil {
					c.config.OnURLsQueued(queue.TotalQueued())
				}
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
			if queue.Add(nextURL, 0) && c.config.OnURLsQueued != nil {
				c.config.OnURLsQueued(queue.TotalQueued())
			}
		}
	}

	logger.Debug("crawler finished processing URL", "url", url)
}
