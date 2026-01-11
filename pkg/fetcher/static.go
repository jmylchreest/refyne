package fetcher

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/refyne/refyne/internal/logger"
)

// StaticConfig holds configuration for the static fetcher.
type StaticConfig struct {
	UserAgent string
	Timeout   time.Duration
}

// DefaultStaticConfig returns sensible defaults.
func DefaultStaticConfig() StaticConfig {
	return StaticConfig{
		UserAgent: defaultUserAgent,
		Timeout:   30 * time.Second,
	}
}

// Chrome user agent for better compatibility
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// StaticFetcher uses Colly for static HTML fetching.
// It implements the Fetcher interface.
type StaticFetcher struct {
	config StaticConfig
}

// NewStatic creates a new static fetcher.
func NewStatic(cfg StaticConfig) *StaticFetcher {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultStaticConfig().UserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultStaticConfig().Timeout
	}
	return &StaticFetcher{config: cfg}
}

// Fetch retrieves page content using Colly.
func (f *StaticFetcher) Fetch(ctx context.Context, targetURL string, opts Options) (Content, error) {
	logger.Debug("static fetch starting", "url", targetURL)

	result := Content{
		URL:       targetURL,
		FetchedAt: time.Now(),
	}

	// Create a new collector for each request
	userAgent := coalesce(opts.UserAgent, f.config.UserAgent)
	c := colly.NewCollector(
		colly.UserAgent(userAgent),
	)
	logger.Debug("static fetch configured", "user_agent", userAgent)

	// Set timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = f.config.Timeout
	}
	c.SetRequestTimeout(timeout)
	logger.Debug("static fetch timeout set", "timeout", timeout)

	// Set custom headers
	if len(opts.Headers) > 0 {
		c.OnRequest(func(r *colly.Request) {
			for k, v := range opts.Headers {
				r.Headers.Set(k, v)
			}
		})
	}

	var fetchErr error

	// Handle response
	c.OnResponse(func(r *colly.Response) {
		result.StatusCode = r.StatusCode
		result.ContentType = r.Headers.Get("Content-Type")
		result.HTML = string(r.Body)
		logger.Debug("static fetch response received",
			"status", r.StatusCode,
			"content_type", result.ContentType,
			"body_size", len(r.Body))
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		statusCode := 0
		if r != nil {
			statusCode = r.StatusCode
			result.StatusCode = statusCode
		}
		fetchErr = fmt.Errorf("fetch error: %w", err)
		logger.Debug("static fetch error", "status", statusCode, "error", err)
	})

	// Perform the request
	logger.Debug("static fetch visiting URL", "url", targetURL)
	if err := c.Visit(targetURL); err != nil {
		logger.Debug("static fetch visit failed", "url", targetURL, "error", err)
		return result, fmt.Errorf("failed to visit URL: %w", err)
	}

	if fetchErr != nil {
		return result, fetchErr
	}

	// Parse HTML and extract content
	if result.HTML != "" {
		logger.Debug("static fetch parsing content", "html_size", len(result.HTML))
		if err := f.parseContent(&result); err != nil {
			logger.Debug("static fetch parse failed", "error", err)
			return result, fmt.Errorf("failed to parse content: %w", err)
		}
		logger.Debug("static fetch parse complete",
			"title", result.Title,
			"text_size", len(result.Text),
			"links_count", len(result.Links))
	}

	logger.Debug("static fetch complete", "url", targetURL)
	return result, nil
}

// parseContent extracts text and metadata from HTML.
func (f *StaticFetcher) parseContent(content *Content) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content.HTML))
	if err != nil {
		return err
	}

	// Extract title
	content.Title = strings.TrimSpace(doc.Find("title").First().Text())

	// Remove script and style elements before extracting text
	doc.Find("script, style, noscript, iframe, svg").Remove()

	// Extract clean text
	var textParts []string
	doc.Find("body").Each(func(_ int, s *goquery.Selection) {
		text := cleanText(s.Text())
		if text != "" {
			textParts = append(textParts, text)
		}
	})
	content.Text = strings.Join(textParts, "\n")

	// Extract links
	baseURL, _ := url.Parse(content.URL)
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || strings.HasPrefix(href, "#") {
			return
		}

		// Resolve relative URLs
		linkURL, err := url.Parse(href)
		if err != nil {
			return
		}
		if !linkURL.IsAbs() && baseURL != nil {
			linkURL = baseURL.ResolveReference(linkURL)
		}

		content.Links = append(content.Links, linkURL.String())
	})

	return nil
}

// Close releases resources.
func (f *StaticFetcher) Close() error {
	return nil
}

// Type returns the fetcher type.
func (f *StaticFetcher) Type() string {
	return "static"
}

// cleanText normalizes whitespace in text.
func cleanText(s string) string {
	// Replace multiple whitespace with single space
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
