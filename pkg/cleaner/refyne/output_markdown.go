package refyne

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// markdownState tracks state during markdown conversion.
type markdownState struct {
	imageCounter int
	images       map[string]ImageRef
	imageOrder   []string
}

// htmlToMarkdown converts a goquery document to markdown format.
// It extracts metadata and optionally prepends frontmatter.
// Images are replaced with placeholders like {{IMG_001}} and cataloged in metadata.
func (c *Cleaner) htmlToMarkdown(doc *goquery.Document, result *Result) string {
	// Initialize state for tracking images during conversion
	state := &markdownState{
		imageCounter: 0,
		images:       make(map[string]ImageRef),
		imageOrder:   []string{},
	}

	// Extract headings separately (they don't need state tracking)
	metadata := &ContentMetadata{
		Images:     make(map[string]ImageRef),
		ImageOrder: []string{},
		Headings:   []HeadingRef{},
	}

	if c.config.ExtractHeadings {
		doc.Find("h1, h2, h3, h4, h5, h6").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				tagName := goquery.NodeName(s)
				level := int(tagName[1] - '0')
				id, _ := s.Attr("id")
				metadata.Headings = append(metadata.Headings, HeadingRef{
					Level: level,
					Text:  text,
					ID:    id,
				})
			}
		})
	}

	// Count links
	metadata.LinksCount = doc.Find("a[href]").Length()

	// Format the document as markdown (this populates state.images)
	var sb strings.Builder
	body := doc.Find("body")
	if body.Length() == 0 {
		body = doc.Selection
	}

	c.formatSelectionWithState(&sb, body, "", 0, state)

	// Transfer images from state to metadata
	metadata.Images = state.images
	metadata.ImageOrder = state.imageOrder
	result.Metadata = metadata

	markdown := c.cleanMarkdownOutput(sb.String())

	// Prepend frontmatter if enabled
	if c.config.IncludeFrontmatter {
		frontmatter := c.buildFrontmatter(metadata)
		return frontmatter + markdown
	}

	return markdown
}


// isPlaceholder checks if a URL is a placeholder image.
func (c *Cleaner) isPlaceholder(src string) bool {
	placeholders := []string{
		"pixel.png", "pixel.gif", "blank.png", "blank.gif",
		"spacer.gif", "spacer.png", "1x1.", "data:image",
	}
	srcLower := strings.ToLower(src)
	for _, p := range placeholders {
		if strings.Contains(srcLower, p) {
			return true
		}
	}
	return false
}

// resolveURL resolves a potentially relative URL against the base URL.
// Only performs resolution if ResolveURLs is true.
func (c *Cleaner) resolveURL(rawURL string) string {
	// Handle protocol-relative URLs (always resolve these as they're invalid without protocol)
	if strings.HasPrefix(rawURL, "//") {
		return "https:" + rawURL
	}

	// If it's already absolute, return as-is
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}

	// If ResolveURLs is disabled, keep relative URLs as-is
	if !c.config.ResolveURLs {
		return rawURL
	}

	// If we have a base URL and ResolveURLs is enabled, resolve against it
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
// The frontmatter is designed to be self-documenting for LLM consumption.
func (c *Cleaner) buildFrontmatter(metadata *ContentMetadata) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("# Content Metadata\n")

	// Images section with placeholder explanation
	if c.config.ExtractImages && len(metadata.Images) > 0 {
		sb.WriteString("# Images are referenced in the body as {{IMG_001}}, {{IMG_002}}, etc.\n")
		sb.WriteString("# Use this map to look up the actual URL and description for each placeholder.\n")
		sb.WriteString("images:\n")

		// Output in order
		for _, id := range metadata.ImageOrder {
			img := metadata.Images[id]
			sb.WriteString(fmt.Sprintf("  %s:\n", id))
			sb.WriteString(fmt.Sprintf("    url: %q\n", img.URL))
			if img.Alt != "" {
				sb.WriteString(fmt.Sprintf("    alt: %q\n", img.Alt))
			}
		}
	}

	// Headings section
	if c.config.ExtractHeadings && len(metadata.Headings) > 0 {
		sb.WriteString("\nheadings:\n")
		for _, h := range metadata.Headings {
			sb.WriteString(fmt.Sprintf("  - level: %d\n", h.Level))
			sb.WriteString(fmt.Sprintf("    text: %q\n", h.Text))
			if h.ID != "" {
				sb.WriteString(fmt.Sprintf("    id: %q\n", h.ID))
			}
		}
	}

	// Links count
	sb.WriteString(fmt.Sprintf("\nlinks_count: %d\n", metadata.LinksCount))

	// Hints section
	if c.config.IncludeHints {
		sb.WriteString("\nhints:\n")
		// Default hints for image placeholders
		if c.config.ExtractImages && len(metadata.Images) > 0 {
			sb.WriteString("  - \"Image placeholders like {{IMG_001}} appear in the body where images belong\"\n")
			sb.WriteString("  - \"Look up each placeholder in the 'images' map above to get the actual URL\"\n")
		}
		// Custom hints
		for _, hint := range c.config.CustomHints {
			sb.WriteString(fmt.Sprintf("  - %q\n", hint))
		}
	}

	sb.WriteString("---\n\n")
	return sb.String()
}

// formatSelectionWithState recursively formats a selection as markdown with state tracking.
func (c *Cleaner) formatSelectionWithState(sb *strings.Builder, sel *goquery.Selection, listPrefix string, depth int, state *markdownState) {
	sel.Contents().Each(func(_ int, s *goquery.Selection) {
		node := s.Nodes[0]

		switch node.Type {
		case html.TextNode:
			text := node.Data
			// Collapse whitespace but preserve single spaces
			text = strings.Join(strings.Fields(text), " ")
			if text != "" {
				sb.WriteString(text)
			}

		case html.ElementNode:
			tagName := goquery.NodeName(s)
			c.formatElementWithState(sb, s, tagName, listPrefix, depth, state)
		}
	})
}

// formatElementWithState handles element-specific markdown formatting with state tracking.
func (c *Cleaner) formatElementWithState(sb *strings.Builder, s *goquery.Selection, tag string, listPrefix string, depth int, state *markdownState) {
	switch tag {
	// Headings
	case "h1":
		c.ensureBlankLine(sb)
		sb.WriteString("# ")
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	case "h2":
		c.ensureBlankLine(sb)
		sb.WriteString("## ")
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	case "h3":
		c.ensureBlankLine(sb)
		sb.WriteString("### ")
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	case "h4":
		c.ensureBlankLine(sb)
		sb.WriteString("#### ")
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	case "h5":
		c.ensureBlankLine(sb)
		sb.WriteString("##### ")
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	case "h6":
		c.ensureBlankLine(sb)
		sb.WriteString("###### ")
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	// Paragraphs and blocks
	case "p":
		c.ensureBlankLine(sb)
		c.formatSelectionWithState(sb, s, "", depth, state)
		c.ensureBlankLine(sb)

	case "div", "section", "article", "main", "header", "footer", "figure", "figcaption":
		// Block elements - just process children
		c.formatSelectionWithState(sb, s, "", depth, state)

	// Line breaks
	case "br":
		sb.WriteString("  \n") // Markdown line break

	// Horizontal rule
	case "hr":
		c.ensureBlankLine(sb)
		sb.WriteString("---")
		c.ensureBlankLine(sb)

	// Bold/Strong
	case "strong", "b":
		sb.WriteString("**")
		c.formatSelectionWithState(sb, s, "", depth, state)
		sb.WriteString("**")

	// Italic/Emphasis
	case "em", "i":
		sb.WriteString("*")
		c.formatSelectionWithState(sb, s, "", depth, state)
		sb.WriteString("*")

	// Code (inline)
	case "code":
		parent := s.Parent()
		if parent.Length() > 0 && goquery.NodeName(parent) == "pre" {
			// Handled by pre
			c.formatSelectionWithState(sb, s, "", depth, state)
		} else {
			sb.WriteString("`")
			c.formatSelectionWithState(sb, s, "", depth, state)
			sb.WriteString("`")
		}

	// Code block
	case "pre":
		c.ensureBlankLine(sb)
		sb.WriteString("```\n")
		sb.WriteString(s.Text())
		sb.WriteString("\n```")
		c.ensureBlankLine(sb)

	// Blockquote
	case "blockquote":
		c.ensureBlankLine(sb)
		var quoteSb strings.Builder
		c.formatSelectionWithState(&quoteSb, s, "", depth, state)
		lines := strings.Split(strings.TrimSpace(quoteSb.String()), "\n")
		for _, line := range lines {
			sb.WriteString("> ")
			sb.WriteString(strings.TrimSpace(line))
			sb.WriteString("\n")
		}

	// Unordered list
	case "ul":
		c.ensureBlankLine(sb)
		s.Children().Each(func(_ int, li *goquery.Selection) {
			if goquery.NodeName(li) == "li" {
				indent := strings.Repeat("  ", depth)
				sb.WriteString(indent)
				sb.WriteString("- ")
				c.formatSelectionWithState(sb, li, indent, depth+1, state)
				sb.WriteString("\n")
			}
		})

	// Ordered list
	case "ol":
		c.ensureBlankLine(sb)
		counter := 1
		s.Children().Each(func(_ int, li *goquery.Selection) {
			if goquery.NodeName(li) == "li" {
				indent := strings.Repeat("  ", depth)
				sb.WriteString(indent)
				fmt.Fprintf(sb, "%d. ", counter)
				c.formatSelectionWithState(sb, li, indent, depth+1, state)
				sb.WriteString("\n")
				counter++
			}
		})

	// Definition list
	case "dl":
		c.ensureBlankLine(sb)
		s.Children().Each(func(_ int, child *goquery.Selection) {
			switch goquery.NodeName(child) {
			case "dt":
				sb.WriteString("**")
				c.formatSelectionWithState(sb, child, "", depth, state)
				sb.WriteString("**\n")
			case "dd":
				sb.WriteString(": ")
				c.formatSelectionWithState(sb, child, "", depth, state)
				sb.WriteString("\n")
			}
		})

	// Links
	case "a":
		href, exists := s.Attr("href")
		if !exists || href == "" || strings.HasPrefix(href, "javascript:") {
			c.formatSelectionWithState(sb, s, "", depth, state)
		} else {
			sb.WriteString("[")
			c.formatSelectionWithState(sb, s, "", depth, state)
			sb.WriteString("](")
			sb.WriteString(c.resolveURL(href))
			sb.WriteString(")")
		}

	// Images - use placeholders when extracting to frontmatter
	case "img":
		c.formatImage(sb, s, state)

	// Tables
	case "table":
		c.formatTableWithState(sb, s, state)

	// Skip these elements entirely
	case "script", "style", "noscript", "svg", "iframe", "form", "input", "button", "select", "textarea", "nav":
		// Don't output anything

	// Span and other inline elements - just process children
	default:
		c.formatSelectionWithState(sb, s, "", depth, state)
	}
}

// formatImage handles image elements, using placeholders when extracting to frontmatter.
func (c *Cleaner) formatImage(sb *strings.Builder, s *goquery.Selection, state *markdownState) {
	// Get image source
	src, _ := s.Attr("src")
	if src == "" || c.isPlaceholder(src) {
		src, _ = s.Attr("data-src")
	}
	if src == "" {
		// Try srcset
		srcset, _ := s.Attr("srcset")
		if srcset != "" {
			src = strings.Fields(srcset)[0]
			src = strings.TrimSuffix(src, ",")
		}
	}

	// Skip if no valid source
	if src == "" || strings.HasPrefix(src, "data:") || c.isPlaceholder(src) {
		return
	}

	src = c.resolveURL(src)
	alt, _ := s.Attr("alt")
	alt = strings.TrimSpace(alt)

	// If extracting to frontmatter with placeholders
	if c.config.ExtractImages && c.config.IncludeFrontmatter && state != nil {
		// Generate placeholder ID
		state.imageCounter++
		placeholderID := fmt.Sprintf("IMG_%03d", state.imageCounter)

		// Store image in state
		state.images[placeholderID] = ImageRef{
			URL: src,
			Alt: alt,
		}
		state.imageOrder = append(state.imageOrder, placeholderID)

		// Write placeholder to body
		sb.WriteString("{{")
		sb.WriteString(placeholderID)
		sb.WriteString("}}")
	} else {
		// Include image inline as standard markdown
		sb.WriteString("![")
		sb.WriteString(alt)
		sb.WriteString("](")
		sb.WriteString(src)
		sb.WriteString(")")
	}
}

// formatTableWithState converts an HTML table to markdown with state tracking.
func (c *Cleaner) formatTableWithState(sb *strings.Builder, table *goquery.Selection, state *markdownState) {
	c.ensureBlankLine(sb)

	var headers []string
	var rows [][]string

	// Extract headers
	table.Find("thead tr th, thead tr td, tr:first-child th").Each(func(_ int, cell *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(cell.Text()))
	})

	// If no thead, try first row
	if len(headers) == 0 {
		table.Find("tr:first-child td").Each(func(_ int, cell *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(cell.Text()))
		})
	}

	// Extract rows
	skipFirst := len(headers) > 0
	table.Find("tbody tr, tr").Each(func(i int, tr *goquery.Selection) {
		if skipFirst && i == 0 && tr.Find("th").Length() > 0 {
			return
		}
		if skipFirst && i == 0 && tr.Parent().Is("thead") {
			return
		}

		var row []string
		tr.Find("td, th").Each(func(_ int, cell *goquery.Selection) {
			row = append(row, strings.TrimSpace(cell.Text()))
		})
		if len(row) > 0 {
			rows = append(rows, row)
		}
	})

	// Write header row
	if len(headers) > 0 {
		sb.WriteString("| ")
		sb.WriteString(strings.Join(headers, " | "))
		sb.WriteString(" |\n")

		// Write separator
		sb.WriteString("|")
		for range headers {
			sb.WriteString(" --- |")
		}
		sb.WriteString("\n")
	}

	// Write data rows
	for _, row := range rows {
		sb.WriteString("| ")
		sb.WriteString(strings.Join(row, " | "))
		sb.WriteString(" |\n")
	}
}


// ensureBlankLine ensures the output ends with a blank line.
func (c *Cleaner) ensureBlankLine(sb *strings.Builder) {
	str := sb.String()
	if len(str) == 0 {
		return
	}
	if !strings.HasSuffix(str, "\n\n") {
		if strings.HasSuffix(str, "\n") {
			sb.WriteString("\n")
		} else {
			sb.WriteString("\n\n")
		}
	}
}

// cleanMarkdownOutput normalizes whitespace in the markdown output.
func (c *Cleaner) cleanMarkdownOutput(s string) string {
	// Replace multiple blank lines with double newline
	lines := strings.Split(s, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		trimmed := strings.TrimRight(line, " \t")
		if strings.TrimSpace(trimmed) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, "")
			}
		} else {
			blankCount = 0
			result = append(result, trimmed)
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}
