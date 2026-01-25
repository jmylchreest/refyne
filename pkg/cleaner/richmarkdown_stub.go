//go:build !markdown

package cleaner

// ImageRef represents an extracted image with context.
type ImageRef struct {
	URL     string `json:"url" yaml:"url"`
	Alt     string `json:"alt,omitempty" yaml:"alt,omitempty"`
	Context string `json:"context,omitempty" yaml:"context,omitempty"`
}

// HeadingRef represents an extracted heading.
type HeadingRef struct {
	Level int    `json:"level" yaml:"level"`
	Text  string `json:"text" yaml:"text"`
	ID    string `json:"id,omitempty" yaml:"id,omitempty"`
}

// ContentMetadata contains structured information extracted from the HTML.
type ContentMetadata struct {
	Images     []ImageRef   `json:"images" yaml:"images"`
	Headings   []HeadingRef `json:"headings" yaml:"headings"`
	LinksCount int          `json:"links_count" yaml:"links_count"`
}

// RichMarkdownConfig configures the rich markdown cleaner.
type RichMarkdownConfig struct {
	IncludeFrontmatter bool
	ExtractImages      bool
	ExtractHeadings    bool
	IncludeHints       bool
	CustomHints        []string
	BaseURL            string
	UseRefyneCleaner   bool
	RefyneConfig       interface{}
}

// RichMarkdownCleaner is a stub that passes through content when markdown is not compiled in.
// Build with -tags markdown to enable the real implementation.
type RichMarkdownCleaner struct {
	config *RichMarkdownConfig
}

// RichMarkdownOption configures the rich markdown cleaner.
type RichMarkdownOption func(*RichMarkdownConfig)

// WithFrontmatter enables or disables frontmatter output.
func WithFrontmatter(enabled bool) RichMarkdownOption {
	return func(c *RichMarkdownConfig) {
		c.IncludeFrontmatter = enabled
	}
}

// WithImageExtraction enables or disables image extraction.
func WithImageExtraction(enabled bool) RichMarkdownOption {
	return func(c *RichMarkdownConfig) {
		c.ExtractImages = enabled
	}
}

// WithHeadingExtraction enables or disables heading extraction.
func WithHeadingExtraction(enabled bool) RichMarkdownOption {
	return func(c *RichMarkdownConfig) {
		c.ExtractHeadings = enabled
	}
}

// WithHints enables or disables LLM hints in frontmatter.
func WithHints(enabled bool) RichMarkdownOption {
	return func(c *RichMarkdownConfig) {
		c.IncludeHints = enabled
	}
}

// WithCustomHints adds custom hints to the frontmatter.
func WithCustomHints(hints ...string) RichMarkdownOption {
	return func(c *RichMarkdownConfig) {
		c.CustomHints = append(c.CustomHints, hints...)
	}
}

// WithBaseURL sets the base URL for resolving relative URLs.
func WithBaseURL(baseURL string) RichMarkdownOption {
	return func(c *RichMarkdownConfig) {
		c.BaseURL = baseURL
	}
}

// NewRichMarkdown returns a stub cleaner when markdown is not compiled in.
// The cleaner passes through content unchanged (equivalent to noop).
func NewRichMarkdown(_ ...RichMarkdownOption) *RichMarkdownCleaner {
	return &RichMarkdownCleaner{
		config: &RichMarkdownConfig{},
	}
}

// Clean passes through content unchanged when markdown is not compiled in.
func (c *RichMarkdownCleaner) Clean(html string) (string, error) {
	return html, nil
}

// Name returns the cleaner type.
func (c *RichMarkdownCleaner) Name() string {
	return "richmarkdown"
}

// IsAvailable returns false when richmarkdown is not compiled in.
func (c *RichMarkdownCleaner) IsAvailable() bool {
	return false
}

// ExtractMetadataOnly returns empty metadata when markdown is not compiled in.
func (c *RichMarkdownCleaner) ExtractMetadataOnly(html string) (ContentMetadata, error) {
	return ContentMetadata{
		Images:   []ImageRef{},
		Headings: []HeadingRef{},
	}, nil
}
