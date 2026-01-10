package scraper

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/refyne/refyne/internal/logger"
)

// DynamicFetcher uses chromedp for JavaScript-rendered pages.
type DynamicFetcher struct {
	config    FetcherConfig
	allocCtx  context.Context
	cancelCtx context.CancelFunc
}

// NewDynamicFetcher creates a new dynamic fetcher with a browser instance.
func NewDynamicFetcher(cfg FetcherConfig) (*DynamicFetcher, error) {
	logger.Debug("creating dynamic fetcher")

	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultFetcherConfig().UserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultFetcherConfig().Timeout
	}

	// Create browser allocator with options
	// Include stealth options to avoid bot detection
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),
		chromedp.UserAgent(cfg.UserAgent),
		// Window size to look like a real browser
		chromedp.WindowSize(1920, 1080),
	)

	logger.Debug("dynamic fetcher browser options configured",
		"user_agent", cfg.UserAgent,
		"timeout", cfg.Timeout)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	logger.Debug("dynamic fetcher browser allocator created")

	return &DynamicFetcher{
		config:    cfg,
		allocCtx:  allocCtx,
		cancelCtx: cancelAlloc,
	}, nil
}

// Fetch retrieves page content using a headless browser.
func (f *DynamicFetcher) Fetch(ctx context.Context, targetURL string, opts FetchOptions) (PageContent, error) {
	logger.Debug("dynamic fetch starting", "url", targetURL)

	result := PageContent{
		URL:       targetURL,
		FetchedAt: time.Now(),
	}

	// Create a new browser context for this request
	logger.Debug("dynamic fetch creating browser context")
	browserCtx, cancelBrowser := chromedp.NewContext(f.allocCtx)
	defer cancelBrowser()

	// Set timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = f.config.Timeout
	}
	logger.Debug("dynamic fetch timeout set", "timeout", timeout)

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, timeout)
	defer cancelTimeout()

	// Build the actions
	var html string
	var title string

	actions := []chromedp.Action{
		chromedp.Navigate(targetURL),
	}
	logger.Debug("dynamic fetch navigating", "url", targetURL)

	// Wait for selector if specified
	waitSelector := "body"
	if opts.WaitForSelector != "" {
		waitSelector = opts.WaitForSelector
		actions = append(actions, chromedp.WaitVisible(opts.WaitForSelector))
	} else {
		// Default: wait for body to be visible
		actions = append(actions, chromedp.WaitVisible("body"))
	}
	logger.Debug("dynamic fetch waiting for selector", "selector", waitSelector)

	// Additional wait if specified
	if opts.WaitDuration > 0 {
		actions = append(actions, chromedp.Sleep(opts.WaitDuration))
		logger.Debug("dynamic fetch additional wait", "duration", opts.WaitDuration)
	}

	// Extract content
	actions = append(actions,
		chromedp.OuterHTML("html", &html),
		chromedp.Title(&title),
	)

	// Execute actions
	logger.Debug("dynamic fetch executing browser actions", "action_count", len(actions))
	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		logger.Debug("dynamic fetch browser automation failed", "url", targetURL, "error", err)
		return result, fmt.Errorf("browser automation failed: %w", err)
	}
	logger.Debug("dynamic fetch browser actions complete", "html_size", len(html), "title", title)

	result.HTML = html
	result.Title = title
	result.StatusCode = 200 // chromedp doesn't easily expose status codes

	// Parse content
	logger.Debug("dynamic fetch parsing content")
	if err := f.parseContent(&result); err != nil {
		logger.Debug("dynamic fetch parse failed", "error", err)
		return result, fmt.Errorf("failed to parse content: %w", err)
	}
	logger.Debug("dynamic fetch parse complete",
		"text_size", len(result.Text),
		"links_count", len(result.Links))

	logger.Debug("dynamic fetch complete", "url", targetURL)
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
