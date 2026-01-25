//go:build markdown

package cleaner

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// MarkdownCleaner converts HTML to Markdown using html-to-markdown.
// This is the recommended cleaner for LLM extraction as it:
// - Preserves semantic structure (headers, lists, tables)
// - Removes scripts, styles, and other non-content elements
// - Produces clean, readable text that LLMs understand well
type MarkdownCleaner struct{}

// MarkdownOption configures the markdown cleaner.
type MarkdownOption func(*markdownConfig)

type markdownConfig struct {
	// StripLinks removes link URLs, keeping only the link text
	StripLinks bool
	// StripImages removes images entirely
	StripImages bool
}

// WithStripLinks configures the cleaner to remove link URLs.
func WithStripLinks(strip bool) MarkdownOption {
	return func(c *markdownConfig) {
		c.StripLinks = strip
	}
}

// WithStripImages configures the cleaner to remove images.
func WithStripImages(strip bool) MarkdownOption {
	return func(c *markdownConfig) {
		c.StripImages = strip
	}
}

// NewMarkdown creates a new Markdown cleaner.
func NewMarkdown(opts ...MarkdownOption) *MarkdownCleaner {
	// Options are reserved for future use (e.g., custom conversion rules)
	return &MarkdownCleaner{}
}

// Clean converts HTML to Markdown.
func (c *MarkdownCleaner) Clean(html string) (string, error) {
	markdown, err := md.ConvertString(html)
	if err != nil {
		return "", err
	}

	// Clean up excessive whitespace
	markdown = cleanWhitespace(markdown)

	return markdown, nil
}

// Name returns the cleaner type.
func (c *MarkdownCleaner) Name() string {
	return "markdown"
}

// IsAvailable returns true when markdown cleaner is compiled in.
func (c *MarkdownCleaner) IsAvailable() bool {
	return true
}

// cleanWhitespace normalizes whitespace in the output.
func cleanWhitespace(s string) string {
	// Replace multiple blank lines with a single blank line (max 2 consecutive newlines)
	lines := strings.Split(s, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			// Allow only 1 blank line (2 consecutive newlines: content\n + blank\n)
			if blankCount <= 1 {
				result = append(result, "")
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}
