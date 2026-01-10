package crawler

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// LinkSelector extracts links from HTML content.
type LinkSelector struct {
	CSSSelector string         // CSS selector for links to follow
	URLPattern  *regexp.Regexp // Regex pattern for URLs to match
}

// NewLinkSelector creates a link selector.
func NewLinkSelector(cssSelector string, urlPattern string) (*LinkSelector, error) {
	ls := &LinkSelector{
		CSSSelector: cssSelector,
	}

	if urlPattern != "" {
		pattern, err := regexp.Compile(urlPattern)
		if err != nil {
			return nil, err
		}
		ls.URLPattern = pattern
	}

	return ls, nil
}

// ExtractLinks extracts matching links from HTML content.
func (ls *LinkSelector) ExtractLinks(html string, baseURL string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	var links []string
	seen := make(map[string]bool)

	selector := ls.CSSSelector
	if selector == "" {
		selector = "a[href]"
	}

	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip fragments and javascript links
		if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		// Parse and resolve the URL
		linkURL, err := url.Parse(href)
		if err != nil {
			return
		}

		// Make absolute if relative
		if !linkURL.IsAbs() {
			linkURL = base.ResolveReference(linkURL)
		}

		// Remove fragment
		linkURL.Fragment = ""
		fullURL := linkURL.String()

		// Check URL pattern if specified
		if ls.URLPattern != nil && !ls.URLPattern.MatchString(fullURL) {
			return
		}

		// Deduplicate
		if seen[fullURL] {
			return
		}
		seen[fullURL] = true

		links = append(links, fullURL)
	})

	return links, nil
}

// PaginationSelector finds the next page link.
type PaginationSelector struct {
	NextSelector string // CSS selector for "next" link
}

// NewPaginationSelector creates a pagination selector.
func NewPaginationSelector(nextSelector string) *PaginationSelector {
	return &PaginationSelector{
		NextSelector: nextSelector,
	}
}

// FindNextPage finds the URL of the next page.
func (ps *PaginationSelector) FindNextPage(html string, baseURL string) (string, bool) {
	if ps.NextSelector == "" {
		return "", false
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", false
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", false
	}

	var nextURL string
	doc.Find(ps.NextSelector).First().Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip fragments and javascript links
		if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		linkURL, err := url.Parse(href)
		if err != nil {
			return
		}

		if !linkURL.IsAbs() {
			linkURL = base.ResolveReference(linkURL)
		}

		nextURL = linkURL.String()
	})

	return nextURL, nextURL != ""
}
