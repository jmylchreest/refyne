package fetcher

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"github.com/refyne/refyne/internal/logger"
	"github.com/refyne/refyne/pkg/fetcher"
)

// DynamicFetcher uses chromedp for JavaScript-rendered pages.
// It supports stealth mode, Googlebot spoofing, and FlareSolverr integration.
type DynamicFetcher struct {
	config       Config
	allocCtx     context.Context
	cancelCtx    context.CancelFunc
	flareSolverr *FlareSolverr

	// Session cache for reusing FlareSolverr sessions per domain.
	// Sessions keep the same browser instance running in FlareSolverr,
	// avoiding repeated Cloudflare challenges.
	sessions   map[string]string // domain -> sessionID
	sessionsMu sync.RWMutex
}

// NewDynamicFetcher creates a new dynamic fetcher with a browser instance.
func NewDynamicFetcher(cfg Config) (*DynamicFetcher, error) {
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultConfig().UserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}

	// Override user-agent for Googlebot mode
	if cfg.Googlebot {
		cfg.UserAgent = GooglebotMobileUserAgent
	}

	// Create browser allocator with options
	var opts []chromedp.ExecAllocatorOption
	if cfg.Stealth {
		// Use stealth options to avoid bot detection
		opts = append(chromedp.DefaultExecAllocatorOptions[:], StealthExecAllocatorOptions()...)
	} else {
		// Use basic headless options
		opts = append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-dev-shm-usage", true),
			chromedp.Flag("disable-blink-features", "AutomationControlled"),
			chromedp.WindowSize(1920, 1080),
		)
	}

	// Add Chrome binary path if found (chromedp's default lookup may miss it)
	if chromePath := FindChromePath(); chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	opts = append(opts, chromedp.UserAgent(cfg.UserAgent))

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	// Create FlareSolverr client if URL is configured
	var fs *FlareSolverr
	if cfg.FlareSolverrURL != "" {
		fs = NewFlareSolverr(cfg.FlareSolverrURL)
	}

	logger.Debug("dynamic fetcher created",
		"stealth", cfg.Stealth,
		"googlebot", cfg.Googlebot,
		"flaresolverr", cfg.FlareSolverrURL != "",
		"timeout", cfg.Timeout)

	return &DynamicFetcher{
		config:       cfg,
		allocCtx:     allocCtx,
		cancelCtx:    cancelAlloc,
		flareSolverr: fs,
		sessions:     make(map[string]string),
	}, nil
}

// getSession returns the session ID for a domain, or empty string if none exists.
func (f *DynamicFetcher) getSession(domain string) string {
	f.sessionsMu.RLock()
	defer f.sessionsMu.RUnlock()
	return f.sessions[domain]
}

// setSession stores the session ID for a domain.
func (f *DynamicFetcher) setSession(domain, sessionID string) {
	f.sessionsMu.Lock()
	defer f.sessionsMu.Unlock()
	f.sessions[domain] = sessionID
}

// getOrCreateSession gets an existing session or creates a new one for the domain.
func (f *DynamicFetcher) getOrCreateSession(ctx context.Context, domain string) (string, error) {
	// Check for existing session
	if sessionID := f.getSession(domain); sessionID != "" {
		return sessionID, nil
	}

	// Create a new session - use domain as session ID for simplicity
	sessionID := strings.ReplaceAll(domain, ".", "-")
	if err := f.flareSolverr.CreateSession(ctx, sessionID); err != nil {
		// Session creation failed - might already exist, or FlareSolverr issue
		// Try to use it anyway (idempotent session creation isn't guaranteed)
		logger.Debug("FlareSolverr session creation returned error, will try anyway", "session", sessionID, "error", err)
	}

	f.setSession(domain, sessionID)
	return sessionID, nil
}

// Fetch retrieves page content using a headless browser.
func (f *DynamicFetcher) Fetch(ctx context.Context, targetURL string, opts fetcher.Options) (fetcher.Content, error) {
	logger.Debug("dynamic fetch",
		"url", targetURL,
		"stealth", f.config.Stealth,
		"flaresolverr", f.flareSolverr != nil)

	// Extract domain for session management
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fetcher.Content{}, fmt.Errorf("invalid URL: %w", err)
	}
	domain := parsedURL.Host

	// If FlareSolverr is configured, use sessions to avoid repeated challenges
	if f.flareSolverr != nil {
		// Get or create a persistent session for this domain
		sessionID, err := f.getOrCreateSession(ctx, domain)
		if err != nil {
			logger.Debug("failed to create FlareSolverr session", "domain", domain, "error", err)
			// Fall through to chromedp
		} else {
			// Use FlareSolverr with the session
			solution, err := f.flareSolverr.Solve(ctx, targetURL, sessionID)
			if err != nil {
				return fetcher.Content{URL: targetURL, FetchedAt: time.Now()}, err
			}

			// FlareSolverr returns the page content directly
			if solution.Response != "" {
				result := fetcher.Content{
					URL:        targetURL,
					FetchedAt:  time.Now(),
					HTML:       solution.Response,
					StatusCode: solution.Status,
				}

				// Detect challenge pages in the response
				if challenge := detectChallengePage("", result.HTML); challenge != "" {
					logger.Warn("challenge page detected in FlareSolverr response", "url", targetURL, "type", challenge)
					return result, fmt.Errorf("%w: %s", fetcher.ErrAntiBot, challenge)
				}

				// Parse content (extract title, text, links)
				if err := f.parseContent(&result); err != nil {
					return result, fmt.Errorf("failed to parse content: %w", err)
				}

				logger.Debug("dynamic fetch complete (via FlareSolverr)",
					"url", targetURL,
					"session", sessionID,
					"title", result.Title,
					"text_size", len(result.Text),
					"links", len(result.Links))

				return result, nil
			}

			// FlareSolverr didn't return content (unusual) - fall through to chromedp
			logger.Debug("FlareSolverr returned no response content, falling back to chromedp")
		}
	}

	// Fetch using browser (no FlareSolverr, or FlareSolverr returned no content)
	return f.fetchWithBrowser(ctx, targetURL, opts)
}

// fetchWithBrowser fetches a page using chromedp with the given options.
func (f *DynamicFetcher) fetchWithBrowser(ctx context.Context, targetURL string, opts fetcher.Options) (fetcher.Content, error) {
	result := fetcher.Content{
		URL:       targetURL,
		FetchedAt: time.Now(),
	}

	logger.Debug("chromedp starting browser context", "url", targetURL)

	// Create a new browser context for this request
	browserCtx, cancelBrowser := chromedp.NewContext(f.allocCtx,
		chromedp.WithLogf(func(format string, args ...interface{}) {
			logger.Debug("chromedp", "msg", fmt.Sprintf(format, args...))
		}),
	)
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
	var actions []chromedp.Action

	// Set cookies before navigation (e.g., cf_clearance from FlareSolverr)
	if len(opts.Cookies) > 0 {
		actions = append(actions, setCookies(targetURL, opts.Cookies))
	}

	if f.config.Stealth {
		// Inject stealth script before navigation to evade bot detection
		actions = append(actions, InjectStealthScript())
	}
	actions = append(actions, chromedp.Navigate(targetURL))

	// Wait for selector if specified
	if opts.WaitForSelector != "" {
		actions = append(actions, chromedp.WaitReady(opts.WaitForSelector))
	} else {
		// Default: wait for body to be ready (WaitVisible has a bug causing infinite polling)
		actions = append(actions, chromedp.WaitReady("body"))
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
	logger.Debug("chromedp executing actions",
		"url", targetURL,
		"action_count", len(actions),
		"timeout", timeout,
		"stealth", f.config.Stealth,
		"cookies", len(opts.Cookies))

	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		// Attempt to capture a debug screenshot on failure
		if screenshot := CaptureScreenshotOnError(browserCtx); screenshot != nil {
			screenshotPath := filepath.Join(os.TempDir(), fmt.Sprintf("refyne-debug-%d.png", time.Now().UnixNano()))
			if writeErr := os.WriteFile(screenshotPath, screenshot, 0644); writeErr == nil {
				logger.Debug("debug screenshot saved", "path", screenshotPath)
			}
		}
		// Check if this looks like a timeout (likely anti-bot blocking)
		if ctx.Err() != nil || strings.Contains(err.Error(), "deadline exceeded") {
			logger.Warn("browser timeout - possible anti-bot protection", "url", targetURL)
			return result, fmt.Errorf("%w: %v", fetcher.ErrChallengeTimeout, err)
		}
		return result, fmt.Errorf("browser automation failed: %w", err)
	}

	result.HTML = html
	result.Title = title
	result.StatusCode = 200 // chromedp doesn't easily expose status codes

	// Detect challenge pages in the response
	if challenge := detectChallengePage(title, html); challenge != "" {
		logger.Warn("challenge page detected", "url", targetURL, "type", challenge)
		return result, fmt.Errorf("%w: %s", fetcher.ErrAntiBot, challenge)
	}

	// Parse content
	if err := f.parseContent(&result); err != nil {
		return result, fmt.Errorf("failed to parse content: %w", err)
	}

	logger.Debug("dynamic fetch complete",
		"url", targetURL,
		"title", title,
		"text_size", len(result.Text),
		"links", len(result.Links))

	return result, nil
}

// detectChallengePage checks if the page content indicates a challenge/CAPTCHA page.
func detectChallengePage(title, html string) string {
	titleLower := strings.ToLower(title)
	htmlLower := strings.ToLower(html)

	// Cloudflare challenges
	if strings.Contains(titleLower, "just a moment") ||
		strings.Contains(titleLower, "attention required") ||
		strings.Contains(htmlLower, "cf-challenge") ||
		strings.Contains(htmlLower, "cf_chl_opt") {
		return "cloudflare"
	}

	// Cloudflare Turnstile
	if strings.Contains(htmlLower, "challenges.cloudflare.com/turnstile") ||
		strings.Contains(htmlLower, "cf-turnstile") {
		return "cloudflare-turnstile"
	}

	// hCaptcha
	if strings.Contains(htmlLower, "hcaptcha.com") ||
		strings.Contains(htmlLower, "h-captcha") {
		return "hcaptcha"
	}

	// reCAPTCHA
	if strings.Contains(htmlLower, "google.com/recaptcha") ||
		strings.Contains(htmlLower, "g-recaptcha") {
		return "recaptcha"
	}

	// Generic bot detection pages
	if strings.Contains(titleLower, "access denied") ||
		strings.Contains(titleLower, "blocked") ||
		strings.Contains(titleLower, "bot detection") ||
		strings.Contains(htmlLower, "robot or human") {
		return "anti-bot"
	}

	return ""
}

// parseContent extracts title, text and links from HTML.
func (f *DynamicFetcher) parseContent(content *fetcher.Content) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content.HTML))
	if err != nil {
		return err
	}

	// Extract title if not already set
	if content.Title == "" {
		content.Title = strings.TrimSpace(doc.Find("title").First().Text())
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

// cleanText normalizes whitespace in text.
func cleanText(s string) string {
	s = strings.TrimSpace(s)
	// Collapse multiple whitespace into single space
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

// Close releases browser resources and destroys FlareSolverr sessions.
func (f *DynamicFetcher) Close() error {
	// Destroy FlareSolverr sessions
	if f.flareSolverr != nil {
		f.sessionsMu.RLock()
		sessions := make([]string, 0, len(f.sessions))
		for _, sessionID := range f.sessions {
			sessions = append(sessions, sessionID)
		}
		f.sessionsMu.RUnlock()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		for _, sessionID := range sessions {
			f.flareSolverr.DestroySession(ctx, sessionID)
		}
	}

	if f.cancelCtx != nil {
		f.cancelCtx()
	}
	return nil
}

// Type returns the fetcher type.
func (f *DynamicFetcher) Type() string {
	return "dynamic"
}

// setCookies returns a chromedp action that sets cookies before navigation.
func setCookies(targetURL string, cookies []fetcher.Cookie) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		u, err := url.Parse(targetURL)
		if err != nil {
			return fmt.Errorf("failed to parse URL for cookies: %w", err)
		}

		var cookieParams []*network.CookieParam
		for _, c := range cookies {
			domain := c.Domain
			if domain == "" {
				domain = u.Host
			}
			cookieParams = append(cookieParams, &network.CookieParam{
				Name:   c.Name,
				Value:  c.Value,
				Domain: domain,
				Path:   "/",
				Secure: u.Scheme == "https",
			})
		}

		return network.SetCookies(cookieParams).Do(ctx)
	})
}
