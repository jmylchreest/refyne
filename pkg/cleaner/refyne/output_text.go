package refyne

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// htmlToText extracts plain text from HTML.
func (c *Cleaner) htmlToText(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		// Fallback: strip tags via regex
		tagRegex := regexp.MustCompile(`<[^>]*>`)
		return strings.TrimSpace(tagRegex.ReplaceAllString(html, ""))
	}

	var buf bytes.Buffer
	doc.Find("body").Each(func(_ int, s *goquery.Selection) {
		buf.WriteString(s.Text())
	})

	text := buf.String()
	if c.config.CollapseWhitespace {
		text = whitespaceRegex.ReplaceAllString(text, " ")
	}
	return strings.TrimSpace(text)
}
