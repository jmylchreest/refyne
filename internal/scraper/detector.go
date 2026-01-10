package scraper

import (
	"context"
	"fmt"
	"strings"
)

// FetchMode determines how pages are fetched.
type FetchMode string

const (
	FetchModeAuto    FetchMode = "auto"
	FetchModeStatic  FetchMode = "static"
	FetchModeDynamic FetchMode = "dynamic"
)

// NewFetcher creates an appropriate fetcher based on mode.
func NewFetcher(mode FetchMode, cfg FetcherConfig) (Fetcher, error) {
	switch mode {
	case FetchModeStatic:
		return NewStaticFetcher(cfg), nil
	case FetchModeDynamic:
		return NewDynamicFetcher(cfg)
	case FetchModeAuto:
		return NewAutoFetcher(cfg)
	default:
		return nil, fmt.Errorf("unknown fetch mode: %s", mode)
	}
}

// AutoFetcher automatically detects whether to use static or dynamic fetching.
type AutoFetcher struct {
	static  *StaticFetcher
	dynamic *DynamicFetcher
	config  FetcherConfig
}

// NewAutoFetcher creates a fetcher that auto-detects JS requirements.
func NewAutoFetcher(cfg FetcherConfig) (*AutoFetcher, error) {
	dynamic, err := NewDynamicFetcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic fetcher: %w", err)
	}

	return &AutoFetcher{
		static:  NewStaticFetcher(cfg),
		dynamic: dynamic,
		config:  cfg,
	}, nil
}

// Fetch tries static first, then falls back to dynamic if needed.
func (f *AutoFetcher) Fetch(ctx context.Context, url string, opts FetchOptions) (PageContent, error) {
	// First, try static fetch
	content, err := f.static.Fetch(ctx, url, opts)
	if err != nil {
		// If static fetch completely failed, try dynamic
		return f.dynamic.Fetch(ctx, url, opts)
	}

	// Check if the page appears to need JavaScript
	if f.needsJavaScript(content) {
		// Retry with dynamic fetcher
		return f.dynamic.Fetch(ctx, url, opts)
	}

	return content, nil
}

// needsJavaScript checks if a page appears to require JS rendering.
func (f *AutoFetcher) needsJavaScript(content PageContent) bool {
	html := strings.ToLower(content.HTML)
	text := strings.ToLower(content.Text)

	// Check for SPA framework markers
	spaMarkers := []string{
		"<div id=\"root\"></div>",       // React
		"<div id=\"app\"></div>",        // Vue
		"<app-root></app-root>",         // Angular
		"<div id=\"__next\"></div>",     // Next.js
		"<div id=\"__nuxt\"></div>",     // Nuxt.js
		"<div data-reactroot",           // React
		"ng-app",                        // Angular
		"v-cloak",                       // Vue
	}

	for _, marker := range spaMarkers {
		if strings.Contains(html, marker) {
			return true
		}
	}

	// Check for very little text content (might be SPA loading)
	if len(strings.TrimSpace(content.Text)) < 100 {
		// Check for JS loading indicators
		jsIndicators := []string{
			"loading",
			"please wait",
			"javascript required",
			"enable javascript",
		}
		for _, indicator := range jsIndicators {
			if strings.Contains(text, indicator) {
				return true
			}
		}
	}

	// Check for noscript tags suggesting JS is needed
	if strings.Contains(html, "<noscript>") {
		noscriptContent := extractBetween(html, "<noscript>", "</noscript>")
		warningIndicators := []string{
			"javascript",
			"enable",
			"required",
			"browser",
		}
		for _, indicator := range warningIndicators {
			if strings.Contains(noscriptContent, indicator) {
				return true
			}
		}
	}

	return false
}

// extractBetween extracts content between two markers.
func extractBetween(s, start, end string) string {
	startIdx := strings.Index(s, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)

	endIdx := strings.Index(s[startIdx:], end)
	if endIdx == -1 {
		return ""
	}

	return s[startIdx : startIdx+endIdx]
}

// Close releases all fetcher resources.
func (f *AutoFetcher) Close() error {
	if f.dynamic != nil {
		return f.dynamic.Close()
	}
	return nil
}

// Type returns the fetcher type.
func (f *AutoFetcher) Type() string {
	return "auto"
}
