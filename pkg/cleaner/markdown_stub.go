//go:build !markdown

package cleaner

// MarkdownCleaner is a stub that passes through content when markdown is not compiled in.
// Build with -tags markdown to enable the real implementation.
type MarkdownCleaner struct{}

// MarkdownOption configures the markdown cleaner.
type MarkdownOption func(*markdownConfig)

type markdownConfig struct {
	StripLinks  bool
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

// NewMarkdown returns a stub cleaner when markdown is not compiled in.
// The cleaner passes through content unchanged (equivalent to noop).
func NewMarkdown(_ ...MarkdownOption) *MarkdownCleaner {
	return &MarkdownCleaner{}
}

// Clean passes through content unchanged when markdown is not compiled in.
func (c *MarkdownCleaner) Clean(html string) (string, error) {
	// Pass through unchanged - equivalent to noop
	return html, nil
}

// Name returns the cleaner type.
func (c *MarkdownCleaner) Name() string {
	return "markdown"
}

// IsAvailable returns false when markdown is not compiled in.
func (c *MarkdownCleaner) IsAvailable() bool {
	return false
}
