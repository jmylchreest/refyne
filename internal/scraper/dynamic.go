package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// DynamicFetcher uses chromedp for JavaScript-rendered pages.
type DynamicFetcher struct {
	config    FetcherConfig
	allocCtx  context.Context
	cancelCtx context.CancelFunc
}

// NewDynamicFetcher creates a new dynamic fetcher with a browser instance.
func NewDynamicFetcher(cfg FetcherConfig) (*DynamicFetcher, error) {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultFetcherConfig().UserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultFetcherConfig().Timeout
	}

	// Create browser allocator with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(cfg.UserAgent),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	return &DynamicFetcher{
		config:    cfg,
		allocCtx:  allocCtx,
		cancelCtx: cancelAlloc,
	}, nil
}

// Fetch retrieves page content using a headless browser.
func (f *DynamicFetcher) Fetch(ctx context.Context, targetURL string, opts FetchOptions) (PageContent, error) {
	result := PageContent{
		URL:       targetURL,
		FetchedAt: time.Now(),
	}

	// Create a new browser context for this request
	browserCtx, cancelBrowser := chromedp.NewContext(f.allocCtx)
	defer cancelBrowser()

	// Set timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = f.config.Timeout
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, timeout)
	defer cancelTimeout()

	// Build the actions
	var html string
	var title string

	actions := []chromedp.Action{
		chromedp.Navigate(targetURL),
	}

	// Wait for selector if specified
	if opts.WaitForSelector != "" {
		actions = append(actions, chromedp.WaitVisible(opts.WaitForSelector))
	} else {
		// Default: wait for body to be visible
		actions = append(actions, chromedp.WaitVisible("body"))
	}

	// Additional wait if specified
	if opts.WaitDuration > 0 {
		actions = append(actions, chromedp.Sleep(opts.WaitDuration))
	}

	// Extract content
	actions = append(actions,
		chromedp.OuterHTML("html", &html),
		chromedp.Title(&title),
	)

	// Execute actions
	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		return result, fmt.Errorf("browser automation failed: %w", err)
	}

	result.HTML = html
	result.Title = title
	result.StatusCode = 200 // chromedp doesn't easily expose status codes

	// Parse content
	if err := f.parseContent(&result); err != nil {
		return result, fmt.Errorf("failed to parse content: %w", err)
	}

	return result, nil
}

// parseContent extracts text and links from HTML.
func (f *DynamicFetcher) parseContent(content *PageContent) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content.HTML))
	if err != nil {
		return err
	}

	// Remove script and style elements
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

// Close releases browser resources.
func (f *DynamicFetcher) Close() error {
	if f.cancelCtx != nil {
		f.cancelCtx()
	}
	return nil
}

// Type returns the fetcher type.
func (f *DynamicFetcher) Type() string {
	return "dynamic"
}
