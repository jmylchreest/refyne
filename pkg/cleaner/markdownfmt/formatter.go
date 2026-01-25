// Package markdownfmt provides a simple, focused HTML to Markdown converter.
// Unlike full-featured converters, this focuses on text formatting while
// allowing the caller to handle structural elements (images, links) separately.
package markdownfmt

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// Config configures the markdown formatter.
type Config struct {
	// SkipImages omits images from output (for separate handling)
	SkipImages bool

	// SkipLinks converts links to plain text (for separate handling)
	SkipLinks bool

	// PreserveLineBreaks keeps <br> as line breaks
	PreserveLineBreaks bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		SkipImages:         true,  // We extract images separately
		SkipLinks:          false, // Keep links by default
		PreserveLineBreaks: true,
	}
}

// Formatter converts HTML to Markdown with full control over output.
type Formatter struct {
	config *Config
}

// New creates a new markdown formatter.
func New(config *Config) *Formatter {
	if config == nil {
		config = DefaultConfig()
	}
	return &Formatter{config: config}
}

// Format converts HTML to Markdown.
func (f *Formatter) Format(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	f.formatNode(&sb, doc.Selection, 0)

	return f.cleanOutput(sb.String()), nil
}

// formatNode recursively formats a selection and its children.
func (f *Formatter) formatNode(sb *strings.Builder, sel *goquery.Selection, depth int) {
	sel.Contents().Each(func(_ int, s *goquery.Selection) {
		node := s.Nodes[0]

		switch node.Type {
		case html.TextNode:
			// Clean and write text content
			text := strings.TrimSpace(node.Data)
			if text != "" {
				// Preserve single spaces between inline elements
				if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") && !strings.HasSuffix(sb.String(), " ") {
					sb.WriteString(" ")
				}
				sb.WriteString(text)
			}

		case html.ElementNode:
			tagName := goquery.NodeName(s)
			f.formatElement(sb, s, tagName, depth)
		}
	})
}

// formatElement handles element-specific formatting.
func (f *Formatter) formatElement(sb *strings.Builder, s *goquery.Selection, tag string, depth int) {
	switch tag {
	// Headings
	case "h1":
		f.ensureNewline(sb, 2)
		sb.WriteString("# ")
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	case "h2":
		f.ensureNewline(sb, 2)
		sb.WriteString("## ")
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	case "h3":
		f.ensureNewline(sb, 2)
		sb.WriteString("### ")
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	case "h4":
		f.ensureNewline(sb, 2)
		sb.WriteString("#### ")
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	case "h5":
		f.ensureNewline(sb, 2)
		sb.WriteString("##### ")
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	case "h6":
		f.ensureNewline(sb, 2)
		sb.WriteString("###### ")
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	// Paragraphs and blocks
	case "p", "div", "section", "article", "main", "header", "footer":
		f.ensureNewline(sb, 2)
		f.formatNode(sb, s, depth)
		f.ensureNewline(sb, 2)

	// Line breaks
	case "br":
		if f.config.PreserveLineBreaks {
			sb.WriteString("  \n") // Markdown line break
		}

	// Horizontal rule
	case "hr":
		f.ensureNewline(sb, 2)
		sb.WriteString("---")
		f.ensureNewline(sb, 2)

	// Bold/Strong
	case "strong", "b":
		sb.WriteString("**")
		f.formatNode(sb, s, depth)
		sb.WriteString("**")

	// Italic/Emphasis
	case "em", "i":
		sb.WriteString("*")
		f.formatNode(sb, s, depth)
		sb.WriteString("*")

	// Code (inline)
	case "code":
		// Check if parent is pre (code block)
		parent := s.Parent()
		if parent.Length() > 0 && goquery.NodeName(parent) == "pre" {
			// Handled by pre
			f.formatNode(sb, s, depth)
		} else {
			sb.WriteString("`")
			f.formatNode(sb, s, depth)
			sb.WriteString("`")
		}

	// Code block
	case "pre":
		f.ensureNewline(sb, 2)
		sb.WriteString("```\n")
		// Get text content directly to preserve whitespace
		sb.WriteString(s.Text())
		sb.WriteString("\n```")
		f.ensureNewline(sb, 2)

	// Blockquote
	case "blockquote":
		f.ensureNewline(sb, 2)
		var quoteSb strings.Builder
		f.formatNode(&quoteSb, s, depth)
		lines := strings.Split(strings.TrimSpace(quoteSb.String()), "\n")
		for _, line := range lines {
			sb.WriteString("> ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		f.ensureNewline(sb, 1)

	// Unordered list
	case "ul":
		f.ensureNewline(sb, 2)
		s.Children().Each(func(_ int, li *goquery.Selection) {
			if goquery.NodeName(li) == "li" {
				sb.WriteString("- ")
				f.formatNode(sb, li, depth+1)
				sb.WriteString("\n")
			}
		})
		f.ensureNewline(sb, 1)

	// Ordered list
	case "ol":
		f.ensureNewline(sb, 2)
		s.Children().Each(func(i int, li *goquery.Selection) {
			if goquery.NodeName(li) == "li" {
				sb.WriteString(strings.Repeat(" ", depth*2))
				sb.WriteString(itoa(i+1) + ". ")
				f.formatNode(sb, li, depth+1)
				sb.WriteString("\n")
			}
		})
		f.ensureNewline(sb, 1)

	// Links
	case "a":
		if f.config.SkipLinks {
			// Just output the text
			f.formatNode(sb, s, depth)
		} else {
			href, exists := s.Attr("href")
			if !exists || href == "" {
				f.formatNode(sb, s, depth)
			} else {
				sb.WriteString("[")
				f.formatNode(sb, s, depth)
				sb.WriteString("](")
				sb.WriteString(href)
				sb.WriteString(")")
			}
		}

	// Images
	case "img":
		if !f.config.SkipImages {
			src, _ := s.Attr("src")
			alt, _ := s.Attr("alt")
			if src != "" {
				sb.WriteString("![")
				sb.WriteString(alt)
				sb.WriteString("](")
				sb.WriteString(src)
				sb.WriteString(")")
			}
		}
		// If skipping images, output nothing

	// Tables
	case "table":
		f.formatTable(sb, s)

	// Skip these elements entirely
	case "script", "style", "noscript", "svg", "iframe", "form", "input", "button", "select", "textarea":
		// Don't output anything

	// Span and other inline elements - just process children
	case "span", "label", "small", "mark", "abbr", "cite", "dfn", "sub", "sup", "time":
		f.formatNode(sb, s, depth)

	// Default: process children
	default:
		f.formatNode(sb, s, depth)
	}
}

// formatTable converts an HTML table to markdown.
func (f *Formatter) formatTable(sb *strings.Builder, table *goquery.Selection) {
	f.ensureNewline(sb, 2)

	var headers []string
	var rows [][]string

	// Extract headers
	table.Find("thead tr, tr").First().Find("th, td").Each(func(_ int, cell *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(cell.Text()))
	})

	// Extract rows (skip first if it was headers)
	isFirst := true
	table.Find("tbody tr, tr").Each(func(_ int, tr *goquery.Selection) {
		if isFirst && tr.Find("th").Length() > 0 {
			isFirst = false
			return
		}
		isFirst = false

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

	f.ensureNewline(sb, 1)
}

// ensureNewline ensures the output ends with the specified number of newlines.
func (f *Formatter) ensureNewline(sb *strings.Builder, count int) {
	str := sb.String()
	trailing := 0
	for i := len(str) - 1; i >= 0 && str[i] == '\n'; i-- {
		trailing++
	}
	for i := trailing; i < count; i++ {
		sb.WriteString("\n")
	}
}

// cleanOutput normalizes whitespace in the output.
func (f *Formatter) cleanOutput(s string) string {
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

// itoa converts int to string (avoid strconv import for simple case)
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}
