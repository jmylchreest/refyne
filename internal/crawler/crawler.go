package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/refyne/refyne/internal/extractor"
	"github.com/refyne/refyne/internal/scraper"
	"github.com/refyne/refyne/pkg/schema"
)

// Result represents a single crawl/extraction result.
type Result struct {
	URL        string
	Data       any
	Raw        string
	Errors     []schema.ValidationError
	Usage      extractor.ExtractionResult
	Error      error
	Depth      int
	FetchedAt  time.Time
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
		Delay:            500 * time.Millisecond,
		Concurrency:      1,
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
	queue := NewURLQueue()
	var linkSelector *LinkSelector
	var paginationSelector *PaginationSelector

	// Setup link selector if configured
	if c.config.FollowSelector != "" || c.config.FollowPattern != "" {
		var err error
		linkSelector, err = NewLinkSelector(c.config.FollowSelector, c.config.FollowPattern)
		if err != nil {
			results <- Result{Error: fmt.Errorf("invalid link selector: %w", err)}
			return
		}
	}

	// Setup pagination selector if configured
	if c.config.NextSelector != "" {
		paginationSelector = NewPaginationSelector(c.config.NextSelector)
	}

	// Add seed URLs to queue at depth 0
	for _, seed := range seeds {
		queue.Add(seed, 0)
	}

	// Track pagination pages
	pagesProcessed := 0

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

		// Check max pages for pagination
		if c.config.MaxPages > 0 && pagesProcessed >= c.config.MaxPages {
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

		pagesProcessed++
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
	// Fetch the page
	content, err := c.fetcher.Fetch(ctx, url, scraper.DefaultFetchOptions())
	if err != nil {
		results <- Result{URL: url, Depth: depth, Error: fmt.Errorf("fetch error: %w", err)}
		return
	}

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
	if shouldExtract {
		extractResult, err := c.extractor.Extract(ctx, content.Text, s)
		if err != nil {
			results <- Result{
				URL:       url,
				Depth:     depth,
				Error:     fmt.Errorf("extraction error: %w", err),
				FetchedAt: content.FetchedAt,
			}
		} else {
			results <- Result{
				URL:       url,
				Depth:     depth,
				Data:      extractResult.Data,
				Raw:       extractResult.Raw,
				Errors:    extractResult.Errors,
				Usage:     extractResult,
				FetchedAt: content.FetchedAt,
			}
		}
	}

	// Follow links if configured and within depth limit
	if linkSelector != nil && depth < c.config.MaxDepth {
		links, err := linkSelector.ExtractLinks(content.HTML, url)
		if err == nil {
			for _, link := range links {
				// Check same domain constraint
				if c.config.SameDomainOnly && !IsSameDomain(url, link) {
					continue
				}
				queue.Add(link, depth+1)
			}
		}
	}

	// Handle pagination (only at depth 0)
	if paginationSelector != nil && depth == 0 {
		if nextURL, found := paginationSelector.FindNextPage(content.HTML, url); found {
			// Pagination stays at depth 0
			queue.Add(nextURL, 0)
		}
	}
}
