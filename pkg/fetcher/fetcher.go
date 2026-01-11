// Package fetcher defines the interface for web page fetching.
// Implement the Fetcher interface to create custom fetchers with specific
// anti-bot evasion, authentication, or other requirements.
package fetcher

import (
	"context"
	"errors"
	"time"
)

// Fetcher abstracts page fetching strategies.
type Fetcher interface {
	// Fetch retrieves page content from a URL.
	Fetch(ctx context.Context, url string, opts Options) (Content, error)

	// Close releases any resources (browser instances, etc.).
	Close() error

	// Type returns a string identifying the fetcher type (e.g., "static", "dynamic").
	Type() string
}

// Options controls fetching behavior.
type Options struct {
	UserAgent       string
	Timeout         time.Duration
	WaitForSelector string        // CSS selector to wait for (dynamic fetchers)
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

// Content represents fetched page data.
type Content struct {
	URL         string
	HTML        string
	Text        string // Extracted readable text
	Title       string
	StatusCode  int
	ContentType string
	FetchedAt   time.Time
	Links       []string // Links found on the page
}

// Error types for distinguishing failure reasons.
// Check with errors.Is(err, fetcher.ErrCaptchaChallenge).
var (
	// ErrCaptchaChallenge indicates the site has an interactive CAPTCHA.
	ErrCaptchaChallenge = errors.New("captcha challenge detected")
	// ErrAntiBot indicates the site's anti-bot protection blocked the request.
	ErrAntiBot = errors.New("anti-bot protection detected")
	// ErrChallengeTimeout indicates a timeout while waiting for challenge to resolve.
	ErrChallengeTimeout = errors.New("challenge timeout")
)
