//go:build markdown

package cleaner

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	md "github.com/JohannesKaufmann/html-to-markdown/v2"

	refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

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
	// IncludeFrontmatter prepends YAML frontmatter with extracted metadata.
	// Default: true
	IncludeFrontmatter bool

	// ExtractImages extracts image URLs and includes them in metadata.
	// Default: true
	ExtractImages bool

	// ExtractHeadings extracts heading structure.
	// Default: true
	ExtractHeadings bool

	// IncludeHints adds LLM hints to the frontmatter.
	// Default: true
	IncludeHints bool

	// CustomHints allows adding custom hints to the frontmatter.
	CustomHints []string

	// BaseURL for resolving relative URLs.
	BaseURL string

	// UseRefyneCleaner enables preprocessing with the refyne cleaner.
	// This handles noscript unwrapping, hidden element removal, etc.
	// Default: true
	UseRefyneCleaner bool

	// RefyneConfig is the configuration for the refyne cleaner.
	// Only used if UseRefyneCleaner is true.
	// If nil, uses refynecleaner.DefaultConfig().
	RefyneConfig *refynecleaner.Config
}

// DefaultRichMarkdownConfig returns the default configuration.
func DefaultRichMarkdownConfig() *RichMarkdownConfig {
	return &RichMarkdownConfig{
		IncludeFrontmatter: true,
		ExtractImages:      true,
		ExtractHeadings:    true,
		IncludeHints:       true,
		UseRefyneCleaner:   true,
		RefyneConfig:       nil, // Will use refynecleaner.DefaultConfig()
	}
}

// RichMarkdownCleaner converts HTML to Markdown with YAML frontmatter
// containing extracted metadata (images, headings, etc.) to help LLMs
// understand the content structure.
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

// NewRichMarkdown creates a new rich markdown cleaner.
func NewRichMarkdown(opts ...RichMarkdownOption) *RichMarkdownCleaner {
	config := DefaultRichMarkdownConfig()
	for _, opt := range opts {
		opt(config)
	}
	return &RichMarkdownCleaner{config: config}
}

// Clean converts HTML to Markdown with frontmatter metadata.
func (c *RichMarkdownCleaner) Clean(html string) (string, error) {
	// Parse HTML for metadata extraction
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract metadata
	metadata := c.extractMetadata(doc)

	// Convert to markdown
	markdown, err := md.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Clean up whitespace
	markdown = cleanWhitespace(markdown)

	// If frontmatter is disabled, just return the markdown
	if !c.config.IncludeFrontmatter {
		return markdown, nil
	}

	// Build frontmatter
	frontmatter := c.buildFrontmatter(metadata)

	// Combine frontmatter and markdown
	return frontmatter + markdown, nil
}

// Name returns the cleaner type.
func (c *RichMarkdownCleaner) Name() string {
	return "richmarkdown"
}

// extractMetadata extracts structured information from the HTML document.
func (c *RichMarkdownCleaner) extractMetadata(doc *goquery.Document) ContentMetadata {
	metadata := ContentMetadata{
		Images:   []ImageRef{},
		Headings: []HeadingRef{},
	}

	// Track current heading context for images
	currentHeading := ""

	// Extract headings and track context
	doc.Find("h1, h2, h3, h4, h5, h6").Each(func(_ int, s *goquery.Selection) {
		if !c.config.ExtractHeadings {
			return
		}

		tagName := goquery.NodeName(s)
		level := int(tagName[1] - '0') // h1 -> 1, h2 -> 2, etc.
		text := strings.TrimSpace(s.Text())
		id, _ := s.Attr("id")

		if text != "" {
			metadata.Headings = append(metadata.Headings, HeadingRef{
				Level: level,
				Text:  text,
				ID:    id,
			})
			currentHeading = text
		}
	})

	// Extract images
	if c.config.ExtractImages {
		// Reset heading context and walk through document in order
		currentHeading = ""
		doc.Find("*").Each(func(_ int, s *goquery.Selection) {
			tagName := goquery.NodeName(s)

			// Update heading context
			if len(tagName) == 2 && tagName[0] == 'h' && tagName[1] >= '1' && tagName[1] <= '6' {
				text := strings.TrimSpace(s.Text())
				if text != "" {
					currentHeading = text
				}
			}

			// Extract images
			if tagName == "img" {
				imgRef := c.extractImageRef(s, currentHeading)
				if imgRef != nil {
					metadata.Images = append(metadata.Images, *imgRef)
				}
			}
		})
	}

	// Count links
	metadata.LinksCount = doc.Find("a[href]").Length()

	return metadata
}

// extractImageRef extracts image information from an img element.
func (c *RichMarkdownCleaner) extractImageRef(s *goquery.Selection, context string) *ImageRef {
	// Try src first, then data-src for lazy-loaded images
	src, exists := s.Attr("src")
	if !exists || src == "" || strings.Contains(src, "pixel.png") || strings.Contains(src, "data:image") {
		// Try data-src for lazy-loaded images
		src, exists = s.Attr("data-src")
		if !exists || src == "" {
			return nil
		}
	}

	// Skip tiny placeholders and data URIs
	if strings.HasPrefix(src, "data:") {
		return nil
	}

	// Resolve relative URLs
	src = c.resolveURL(src)

	// Skip empty or invalid URLs
	if src == "" {
		return nil
	}

	alt, _ := s.Attr("alt")

	return &ImageRef{
		URL:     src,
		Alt:     strings.TrimSpace(alt),
		Context: context,
	}
}

// resolveURL resolves a potentially relative URL against the base URL.
func (c *RichMarkdownCleaner) resolveURL(rawURL string) string {
	// Handle protocol-relative URLs
	if strings.HasPrefix(rawURL, "//") {
		return "https:" + rawURL
	}

	// If it's already absolute, return as-is
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}

	// If we have a base URL, resolve against it
	if c.config.BaseURL != "" {
		base, err := url.Parse(c.config.BaseURL)
		if err == nil {
			ref, err := url.Parse(rawURL)
			if err == nil {
				return base.ResolveReference(ref).String()
			}
		}
	}

	return rawURL
}

// buildFrontmatter creates YAML frontmatter from metadata.
func (c *RichMarkdownCleaner) buildFrontmatter(metadata ContentMetadata) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("# Content metadata extracted for LLM context\n")

	// Images section
	if c.config.ExtractImages && len(metadata.Images) > 0 {
		sb.WriteString("images:\n")
		for _, img := range metadata.Images {
			sb.WriteString(fmt.Sprintf("  - url: %q\n", img.URL))
			if img.Alt != "" {
				sb.WriteString(fmt.Sprintf("    alt: %q\n", img.Alt))
			}
			if img.Context != "" {
				sb.WriteString(fmt.Sprintf("    context: %q\n", img.Context))
			}
		}
	}

	// Headings section
	if c.config.ExtractHeadings && len(metadata.Headings) > 0 {
		sb.WriteString("headings:\n")
		for _, h := range metadata.Headings {
			sb.WriteString(fmt.Sprintf("  - level: %d\n", h.Level))
			sb.WriteString(fmt.Sprintf("    text: %q\n", h.Text))
			if h.ID != "" {
				sb.WriteString(fmt.Sprintf("    id: %q\n", h.ID))
			}
		}
	}

	// Links count
	sb.WriteString(fmt.Sprintf("links_count: %d\n", metadata.LinksCount))

	// Hints section
	if c.config.IncludeHints {
		sb.WriteString("hints:\n")
		// Default hints
		if c.config.ExtractImages && len(metadata.Images) > 0 {
			sb.WriteString("  - \"Image URLs are listed above in the 'images' section - use these URLs directly\"\n")
			sb.WriteString("  - \"Each image's 'context' field indicates which section/step it belongs to\"\n")
		}
		// Custom hints
		for _, hint := range c.config.CustomHints {
			sb.WriteString(fmt.Sprintf("  - %q\n", hint))
		}
	}

	sb.WriteString("---\n\n")
	return sb.String()
}

// ExtractMetadataOnly extracts metadata without converting to markdown.
// Useful when you need just the structured data.
func (c *RichMarkdownCleaner) ExtractMetadataOnly(html string) (ContentMetadata, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ContentMetadata{}, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return c.extractMetadata(doc), nil
}

// IsAvailable returns true when richmarkdown cleaner is compiled in.
func (c *RichMarkdownCleaner) IsAvailable() bool {
	return true
}
