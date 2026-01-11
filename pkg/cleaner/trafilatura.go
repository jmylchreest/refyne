package cleaner

import (
	"bytes"
	"strings"

	"github.com/markusmobius/go-trafilatura"
	"github.com/yosssi/gohtml"
	"golang.org/x/net/html"
)

// OutputFormat specifies Trafilatura output format.
type OutputFormat int

const (
	// OutputHTML outputs cleaned HTML (default, for chaining with MarkdownCleaner).
	OutputHTML OutputFormat = iota
	// OutputText outputs plain text directly.
	OutputText
)

// Toggle specifies include/exclude behavior.
type Toggle int

const (
	// Default uses the default behavior for the field.
	Default Toggle = iota
	// Include explicitly includes the content.
	Include
	// Exclude explicitly excludes the content.
	Exclude
)

// TrafilaturaConfig configures the Trafilatura cleaner.
type TrafilaturaConfig struct {
	// Output format: OutputHTML (default) or OutputText
	Output OutputFormat
	// Comments: Include or Exclude (default: Exclude)
	Comments Toggle
	// Tables: Include or Exclude (default: Include)
	Tables Toggle
	// Links: Include or Exclude (default: Include)
	Links Toggle
	// Images: Include or Exclude (default: Include)
	Images Toggle
	// Fallback to Readability/DomDistiller: Include or Exclude (default: Include)
	Fallback Toggle
}

// TrafilaturaCleaner extracts main content from web pages using go-trafilatura.
// This removes boilerplate content like navigation, ads, headers, and footers,
// leaving only the primary article/content.
type TrafilaturaCleaner struct {
	opts   trafilatura.Options
	output OutputFormat
}

// NewTrafilatura creates a new Trafilatura cleaner.
// Pass nil for default configuration.
func NewTrafilatura(cfg *TrafilaturaConfig) *TrafilaturaCleaner {
	if cfg == nil {
		cfg = &TrafilaturaConfig{}
	}

	// Apply defaults: Comments excluded, everything else included
	excludeComments := cfg.Comments != Include         // default: exclude (unless Include)
	excludeTables := cfg.Tables == Exclude             // default: include (unless Exclude)
	includeLinks := cfg.Links != Exclude               // default: include (unless Exclude)
	includeImages := cfg.Images != Exclude             // default: include (unless Exclude)
	enableFallback := cfg.Fallback != Exclude          // default: enable (unless Exclude)

	return &TrafilaturaCleaner{
		opts: trafilatura.Options{
			ExcludeComments: excludeComments,
			ExcludeTables:   excludeTables,
			IncludeLinks:    includeLinks,
			IncludeImages:   includeImages,
			EnableFallback:  enableFallback,
		},
		output: cfg.Output,
	}
}

// Clean extracts the main content from HTML.
// Returns cleaned HTML or plain text depending on the configured output format.
func (c *TrafilaturaCleaner) Clean(htmlContent string) (string, error) {
	result, err := trafilatura.Extract(strings.NewReader(htmlContent), c.opts)
	if err != nil {
		return "", err
	}

	if result == nil {
		// No content extracted, return original
		return htmlContent, nil
	}

	// Return based on output format
	switch c.output {
	case OutputText:
		if result.ContentText == "" {
			return htmlContent, nil
		}
		return result.ContentText, nil

	case OutputHTML:
		fallthrough
	default:
		if result.ContentNode == nil {
			// Fall back to plain text if no HTML node
			if result.ContentText != "" {
				return result.ContentText, nil
			}
			return htmlContent, nil
		}

		// Render the content node back to HTML for potential chaining
		var buf bytes.Buffer
		if err := html.Render(&buf, result.ContentNode); err != nil {
			// Fall back to plain text if rendering fails
			return result.ContentText, nil
		}

		// Format the HTML for readability
		return gohtml.Format(buf.String()), nil
	}
}

// Name returns the cleaner type.
func (c *TrafilaturaCleaner) Name() string {
	return "trafilatura"
}
