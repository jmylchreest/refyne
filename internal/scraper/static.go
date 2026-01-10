package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// StaticFetcher uses Colly for static HTML fetching.
type StaticFetcher struct {
	config FetcherConfig
}

// NewStaticFetcher creates a new static fetcher.
func NewStaticFetcher(cfg FetcherConfig) *StaticFetcher {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultFetcherConfig().UserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultFetcherConfig().Timeout
	}
	return &StaticFetcher{config: cfg}
}

// Fetch retrieves page content using Colly.
func (f *StaticFetcher) Fetch(ctx context.Context, targetURL string, opts FetchOptions) (PageContent, error) {
	result := PageContent{
		URL:       targetURL,
		FetchedAt: time.Now(),
	}

	// Create a new collector for each request
	c := colly.NewCollector(
		colly.UserAgent(coalesce(opts.UserAgent, f.config.UserAgent)),
	)

	// Set timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = f.config.Timeout
	}
	c.SetRequestTimeout(timeout)

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
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		if r != nil {
			result.StatusCode = r.StatusCode
		}
		fetchErr = fmt.Errorf("fetch error: %w", err)
	})

	// Perform the request
	if err := c.Visit(targetURL); err != nil {
		return result, fmt.Errorf("failed to visit URL: %w", err)
	}

	if fetchErr != nil {
		return result, fetchErr
	}

	// Parse HTML and extract content
	if result.HTML != "" {
		if err := f.parseContent(&result); err != nil {
			return result, fmt.Errorf("failed to parse content: %w", err)
		}
	}

	return result, nil
}

// parseContent extracts text and metadata from HTML.
func (f *StaticFetcher) parseContent(content *PageContent) error {
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
