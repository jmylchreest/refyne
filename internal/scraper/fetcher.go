// Package scraper handles web page fetching and content extraction.
package scraper

import (
	"context"
	"time"
)

// PageContent represents fetched page data.
type PageContent struct {
	URL         string
	HTML        string
	Text        string // Extracted readable text
	Title       string
	StatusCode  int
	ContentType string
	FetchedAt   time.Time
	Links       []string // Links found on the page
}

// FetchOptions controls fetching behavior.
type FetchOptions struct {
	UserAgent       string
	Timeout         time.Duration
	WaitForSelector string        // CSS selector to wait for (dynamic only)
	WaitDuration    time.Duration // Additional wait after load
	Headers         map[string]string
	Cookies         []Cookie
}

// Cookie represents an HTTP cookie.
type Cookie struct {
	Name   string
	Value  string
	Domain string
}

// DefaultFetchOptions returns sensible defaults.
func DefaultFetchOptions() FetchOptions {
	return FetchOptions{
		UserAgent: "refyne/1.0 (https://github.com/refyne/refyne)",
		Timeout:   30 * time.Second,
	}
}

// Fetcher abstracts page fetching strategies.
type Fetcher interface {
	// Fetch retrieves page content from a URL.
	Fetch(ctx context.Context, url string, opts FetchOptions) (PageContent, error)

	// Close releases any resources (browser instances, etc.).
	Close() error

	// Type returns "static" or "dynamic".
	Type() string
}

// FetcherConfig holds common fetcher configuration.
type FetcherConfig struct {
	UserAgent string
	Timeout   time.Duration
}

// DefaultFetcherConfig returns sensible defaults.
func DefaultFetcherConfig() FetcherConfig {
	return FetcherConfig{
		UserAgent: "refyne/1.0 (https://github.com/refyne/refyne)",
		Timeout:   30 * time.Second,
	}
}
