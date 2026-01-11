// Package fetcher provides advanced web fetching implementations for the CLI.
// This includes chromedp-based dynamic fetching with stealth mode, Googlebot
// spoofing, and FlareSolverr integration.
package fetcher

import (
	"time"
)

// Config holds configuration for the dynamic fetcher.
type Config struct {
	UserAgent       string
	Timeout         time.Duration
	Stealth         bool   // Enable anti-bot detection evasion
	Googlebot       bool   // Spoof Googlebot user-agent
	FlareSolverrURL string // FlareSolverr API URL for Cloudflare bypass
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		UserAgent: defaultUserAgent,
		Timeout:   30 * time.Second,
	}
}

// Chrome user agent for better compatibility
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// GooglebotUserAgent is the standard Googlebot user-agent.
const GooglebotUserAgent = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

// GooglebotMobileUserAgent for mobile-first indexing.
const GooglebotMobileUserAgent = "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5X Build/MMB29P) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
