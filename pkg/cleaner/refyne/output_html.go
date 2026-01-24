package refyne

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// htmlOutput generates the final HTML output with configured post-processing.
func (c *Cleaner) htmlOutput(doc *goquery.Document) (string, error) {
	// Get HTML from body (skip the wrapper goquery adds)
	html, err := doc.Find("body").Html()
	if err != nil {
		// Fallback to full document
		html, err = doc.Html()
		if err != nil {
			return "", err
		}
	}

	// Remove comments via regex (since goquery doesn't handle them well)
	if c.config.StripComments {
		html = commentRegex.ReplaceAllString(html, "")
	}

	// Collapse whitespace
	if c.config.CollapseWhitespace {
		html = whitespaceRegex.ReplaceAllString(html, " ")
	}

	// Trim
	if c.config.TrimElements {
		html = strings.TrimSpace(html)
	}

	return html, nil
}
