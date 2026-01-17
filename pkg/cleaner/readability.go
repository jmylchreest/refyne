package cleaner

import (
	"bytes"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"github.com/yosssi/gohtml"
	"golang.org/x/net/html"
)

// ReadabilityConfig configures the Readability cleaner.
type ReadabilityConfig struct {
	// Output format: OutputHTML (default) or OutputText
	Output OutputFormat
	// MaxElemsToParse limits the number of nodes to parse (0 = no limit).
	MaxElemsToParse int
	// NTopCandidates is the number of top candidates to consider (default: 5).
	NTopCandidates int
	// CharThreshold is the minimum character count for valid content (default: 500).
	CharThreshold int
	// KeepClasses preserves CSS classes on elements when true.
	KeepClasses bool
	// ClassesToPreserve specifies specific CSS classes to keep (even if KeepClasses is false).
	ClassesToPreserve []string
	// BaseURL is used for resolving relative URLs. If empty, URLs remain relative.
	BaseURL string
}

// ReadabilityCleaner extracts main content from web pages using go-readability.
// Based on Mozilla's Readability.js, this removes boilerplate while preserving
// images, links, and tables that are part of the main content.
type ReadabilityCleaner struct {
	cfg    ReadabilityConfig
	parser readability.Parser
}

// NewReadability creates a new Readability cleaner.
// Pass nil for default configuration.
func NewReadability(cfg *ReadabilityConfig) *ReadabilityCleaner {
	if cfg == nil {
		cfg = &ReadabilityConfig{}
	}

	parser := readability.NewParser()

	// Apply config options
	if cfg.MaxElemsToParse > 0 {
		parser.MaxElemsToParse = cfg.MaxElemsToParse
	}
	if cfg.NTopCandidates > 0 {
		parser.NTopCandidates = cfg.NTopCandidates
	}
	if cfg.CharThreshold > 0 {
		parser.CharThresholds = cfg.CharThreshold
	}
	if cfg.KeepClasses {
		parser.KeepClasses = true
	}
	if len(cfg.ClassesToPreserve) > 0 {
		parser.ClassesToPreserve = cfg.ClassesToPreserve
	}

	return &ReadabilityCleaner{
		cfg:    *cfg,
		parser: parser,
	}
}

// Clean extracts the main content from HTML using Readability.
// Returns cleaned HTML or plain text depending on the configured output format.
func (c *ReadabilityCleaner) Clean(htmlContent string) (string, error) {
	// Parse the base URL if provided
	var baseURL *url.URL
	if c.cfg.BaseURL != "" {
		var err error
		baseURL, err = url.Parse(c.cfg.BaseURL)
		if err != nil {
			// If URL parsing fails, continue without a base URL
			baseURL = nil
		}
	}

	// Parse the HTML content
	article, err := c.parser.Parse(strings.NewReader(htmlContent), baseURL)
	if err != nil {
		return "", err
	}

	// Check if we got any content
	if article.Node == nil {
		// No content extracted, return original
		return htmlContent, nil
	}

	// Return based on output format
	switch c.cfg.Output {
	case OutputText:
		var buf bytes.Buffer
		if err := article.RenderText(&buf); err != nil {
			return htmlContent, nil
		}
		text := buf.String()
		if text == "" {
			return htmlContent, nil
		}
		return text, nil

	case OutputHTML:
		fallthrough
	default:
		var buf bytes.Buffer
		if err := article.RenderHTML(&buf); err != nil {
			// Fall back to rendering the node directly
			var nodeBuf bytes.Buffer
			if err := html.Render(&nodeBuf, article.Node); err != nil {
				return htmlContent, nil
			}
			return gohtml.Format(nodeBuf.String()), nil
		}
		result := buf.String()
		if result == "" {
			return htmlContent, nil
		}
		// Format the HTML for readability
		return gohtml.Format(result), nil
	}
}

// Name returns the cleaner type.
func (c *ReadabilityCleaner) Name() string {
	return "readability"
}
