package refyne

import (
	"bytes"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Cleaner is a highly configurable HTML content cleaner.
// It implements the cleaner.Cleaner interface.
type Cleaner struct {
	config *Config
	stats  *Stats
}

// New creates a new Cleaner with the given configuration.
// If config is nil, DefaultConfig() is used.
func New(config *Config) *Cleaner {
	if config == nil {
		config = DefaultConfig()
	}
	return &Cleaner{
		config: config,
	}
}

// Name returns the cleaner name for logging.
func (c *Cleaner) Name() string {
	return "refyne"
}

// Clean transforms HTML content according to the configuration.
// This method implements the cleaner.Cleaner interface.
func (c *Cleaner) Clean(html string) (string, error) {
	result := c.CleanWithStats(html)
	if result.Error != nil {
		// Return original content on error (graceful degradation)
		return result.Content, nil
	}
	return result.Content, nil
}

// CleanWithStats performs cleaning and returns detailed stats.
func (c *Cleaner) CleanWithStats(html string) *Result {
	startTime := time.Now()
	result := &Result{
		Stats: NewStats(),
	}
	result.Stats.InputBytes = len(html)

	// Parse HTML
	parseStart := time.Now()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	result.Stats.ParseDuration = time.Since(parseStart)

	if err != nil {
		// Graceful degradation: return original content with warning
		result.Content = html
		result.AddWarning("parse", "HTML parse failed, returning original", err.Error())
		result.Stats.OutputBytes = len(html)
		result.Stats.TotalDuration = time.Since(startTime)
		return result
	}

	// Transform
	transformStart := time.Now()
	c.transform(doc, result)
	result.Stats.TransformDuration = time.Since(transformStart)

	// Generate output
	outputStart := time.Now()
	output, err := c.generateOutput(doc, result)
	result.Stats.OutputDuration = time.Since(outputStart)

	if err != nil {
		result.Content = html
		result.AddWarning("output", "Output generation failed, returning original", err.Error())
		result.Stats.OutputBytes = len(html)
	} else {
		result.Content = output
		result.Stats.OutputBytes = len(output)
	}

	result.Stats.TotalDuration = time.Since(startTime)
	c.stats = result.Stats

	return result
}

// Stats returns the stats from the last Clean operation.
func (c *Cleaner) Stats() *Stats {
	return c.stats
}

// transform applies all configured transformations to the document.
func (c *Cleaner) transform(doc *goquery.Document, result *Result) {
	// Order matters: remove large chunks first, then clean attributes

	// 1. Remove elements by selector (user-defined, most specific)
	if len(c.config.RemoveSelectors) > 0 {
		c.removeBySelectors(doc, result)
	}

	// 2. Remove script/style/comment elements
	if c.config.StripScripts {
		c.removeElements(doc, "script", result)
	}
	if c.config.StripStyles {
		c.removeElements(doc, "style", result)
		c.removeStyleAttributes(doc, result)
	}
	if c.config.StripNoscript {
		c.removeElements(doc, "noscript", result)
	}
	if c.config.StripComments {
		c.removeComments(doc, result)
	}

	// 3. Remove other element types
	if c.config.StripSVGContent {
		c.removeElements(doc, "svg", result)
	}
	if c.config.StripIframes {
		c.removeElements(doc, "iframe", result)
	}

	// 4. Remove hidden elements (before empty check)
	if c.config.StripHiddenElements {
		c.removeHiddenElements(doc, result)
	}

	// 5. Remove event handlers
	if c.config.StripEventHandlers {
		c.removeEventHandlers(doc, result)
	}

	// 6. Clean attributes
	c.cleanAttributes(doc, result)

	// 7. Heuristic-based removals
	if c.config.RemoveByLinkDensity {
		c.removeByLinkDensity(doc, result)
	}
	if c.config.RemoveShortText {
		c.removeShortText(doc, result)
	}

	// 8. Remove empty elements (after other removals)
	if c.config.StripEmptyElements {
		c.removeEmptyElements(doc, result)
	}

	// 9. Whitespace normalization
	if c.config.CollapseWhitespace {
		c.collapseWhitespace(doc, result)
	}

	// Count remaining elements
	doc.Find("*").Each(func(_ int, _ *goquery.Selection) {
		result.Stats.ElementsKept++
	})
}

// removeElements removes all elements matching the given tag.
func (c *Cleaner) removeElements(doc *goquery.Document, tag string, result *Result) {
	doc.Find(tag).Each(func(_ int, s *goquery.Selection) {
		result.Stats.RecordRemoval(tag)
		s.Remove()
	})
}

// removeBySelectors removes elements matching user-defined selectors.
func (c *Cleaner) removeBySelectors(doc *goquery.Document, result *Result) {
	for _, selector := range c.config.RemoveSelectors {
		// Check if this element should be kept
		selection := doc.Find(selector)
		count := selection.Length()
		if count > 0 {
			result.Stats.RecordSelectorMatch(selector, count)
			selection.Each(func(_ int, s *goquery.Selection) {
				// Check against keep selectors
				if !c.shouldKeep(s) {
					tagName := goquery.NodeName(s)
					result.Stats.RecordRemoval(tagName)
					s.Remove()
				}
			})
		}
	}
}

// shouldKeep checks if an element matches any keep selectors.
func (c *Cleaner) shouldKeep(s *goquery.Selection) bool {
	if len(c.config.KeepSelectors) == 0 {
		return false
	}
	for _, selector := range c.config.KeepSelectors {
		if s.Is(selector) {
			return true
		}
	}
	return false
}

// removeStyleAttributes removes style="" attributes from all elements.
func (c *Cleaner) removeStyleAttributes(doc *goquery.Document, result *Result) {
	doc.Find("[style]").Each(func(_ int, s *goquery.Selection) {
		s.RemoveAttr("style")
		result.Stats.AttributesRemoved++
	})
}

// removeComments removes HTML comments from the document.
func (c *Cleaner) removeComments(doc *goquery.Document, result *Result) {
	// goquery doesn't directly expose comments, so we need to work with the underlying nodes
	// For now, we'll handle this in the output phase or use a regex approach
	// This is a limitation we'll note
	result.AddWarning("transform", "Comment removal via DOM not fully implemented", "using regex fallback")
}

// removeHiddenElements removes elements with display:none or hidden attribute.
func (c *Cleaner) removeHiddenElements(doc *goquery.Document, result *Result) {
	// Elements with hidden attribute
	doc.Find("[hidden]").Each(func(_ int, s *goquery.Selection) {
		if !c.shouldKeep(s) {
			result.Stats.HiddenElementRemovals++
			result.Stats.RecordRemoval(goquery.NodeName(s))
			s.Remove()
		}
	})

	// Elements with aria-hidden="true"
	doc.Find("[aria-hidden='true']").Each(func(_ int, s *goquery.Selection) {
		if !c.shouldKeep(s) {
			result.Stats.HiddenElementRemovals++
			result.Stats.RecordRemoval(goquery.NodeName(s))
			s.Remove()
		}
	})

	// Elements with display:none in style attribute
	doc.Find("[style*='display']").Each(func(_ int, s *goquery.Selection) {
		style, exists := s.Attr("style")
		if exists && strings.Contains(strings.ToLower(style), "display:none") ||
			strings.Contains(strings.ToLower(style), "display: none") {
			if !c.shouldKeep(s) {
				result.Stats.HiddenElementRemovals++
				result.Stats.RecordRemoval(goquery.NodeName(s))
				s.Remove()
			}
		}
	})

	// Elements with visibility:hidden in style attribute
	doc.Find("[style*='visibility']").Each(func(_ int, s *goquery.Selection) {
		style, exists := s.Attr("style")
		if exists && strings.Contains(strings.ToLower(style), "visibility:hidden") ||
			strings.Contains(strings.ToLower(style), "visibility: hidden") {
			if !c.shouldKeep(s) {
				result.Stats.HiddenElementRemovals++
				result.Stats.RecordRemoval(goquery.NodeName(s))
				s.Remove()
			}
		}
	})
}

// removeEventHandlers removes onclick, onload, and other event attributes.
func (c *Cleaner) removeEventHandlers(doc *goquery.Document, result *Result) {
	eventAttrs := []string{
		"onclick", "ondblclick", "onmousedown", "onmouseup", "onmouseover",
		"onmousemove", "onmouseout", "onmouseenter", "onmouseleave",
		"onkeydown", "onkeypress", "onkeyup",
		"onload", "onunload", "onabort", "onerror",
		"onfocus", "onblur", "onchange", "onsubmit", "onreset",
		"onscroll", "onresize",
	}

	for _, attr := range eventAttrs {
		doc.Find("[" + attr + "]").Each(func(_ int, s *goquery.Selection) {
			s.RemoveAttr(attr)
			result.Stats.AttributesRemoved++
		})
	}
}

// cleanAttributes removes attributes based on config.
func (c *Cleaner) cleanAttributes(doc *goquery.Document, result *Result) {
	if c.config.StripDataAttributes {
		// Find elements with data-* attributes
		doc.Find("*").Each(func(_ int, s *goquery.Selection) {
			for _, attr := range s.Nodes[0].Attr {
				if strings.HasPrefix(attr.Key, "data-") {
					s.RemoveAttr(attr.Key)
					result.Stats.AttributesRemoved++
				}
			}
		})
	}

	if c.config.StripARIA {
		doc.Find("*").Each(func(_ int, s *goquery.Selection) {
			for _, attr := range s.Nodes[0].Attr {
				if strings.HasPrefix(attr.Key, "aria-") {
					s.RemoveAttr(attr.Key)
					result.Stats.AttributesRemoved++
				}
			}
		})
	}

	if c.config.StripClasses {
		doc.Find("[class]").Each(func(_ int, s *goquery.Selection) {
			s.RemoveAttr("class")
			result.Stats.AttributesRemoved++
		})
	}

	if c.config.StripIDs {
		doc.Find("[id]").Each(func(_ int, s *goquery.Selection) {
			s.RemoveAttr("id")
			result.Stats.AttributesRemoved++
		})
	}

	if c.config.StripMicrodata {
		microdataAttrs := []string{"itemscope", "itemprop", "itemtype", "itemid", "itemref"}
		for _, attr := range microdataAttrs {
			doc.Find("[" + attr + "]").Each(func(_ int, s *goquery.Selection) {
				s.RemoveAttr(attr)
				result.Stats.AttributesRemoved++
			})
		}
	}
}

// removeByLinkDensity removes elements with high link-to-text ratio.
func (c *Cleaner) removeByLinkDensity(doc *goquery.Document, result *Result) {
	threshold := c.config.LinkDensityThreshold
	if threshold <= 0 {
		threshold = 0.5
	}

	// Check block-level elements
	blockElements := "div, section, aside, article, nav, header, footer, p"
	doc.Find(blockElements).Each(func(_ int, s *goquery.Selection) {
		if c.shouldKeep(s) {
			return
		}

		totalText := strings.TrimSpace(s.Text())
		if len(totalText) == 0 {
			return
		}

		linkText := ""
		s.Find("a").Each(func(_ int, a *goquery.Selection) {
			linkText += strings.TrimSpace(a.Text())
		})

		density := float64(len(linkText)) / float64(len(totalText))
		if density > threshold {
			result.Stats.LinkDensityRemovals++
			result.Stats.RecordRemoval(goquery.NodeName(s))
			s.Remove()
		}
	})
}

// removeShortText removes elements with very little text content.
func (c *Cleaner) removeShortText(doc *goquery.Document, result *Result) {
	minLength := c.config.MinTextLength
	if minLength <= 0 {
		minLength = 20
	}

	// Only check leaf-ish elements, not containers
	doc.Find("p, span, div").Each(func(_ int, s *goquery.Selection) {
		if c.shouldKeep(s) {
			return
		}

		// Skip if it has meaningful children
		if s.Find("img, table, ul, ol, form").Length() > 0 {
			return
		}

		text := strings.TrimSpace(s.Text())
		if len(text) < minLength && len(text) > 0 {
			result.Stats.ShortTextRemovals++
			result.Stats.RecordRemoval(goquery.NodeName(s))
			s.Remove()
		}
	})
}

// removeEmptyElements removes elements with no text or meaningful children.
func (c *Cleaner) removeEmptyElements(doc *goquery.Document, result *Result) {
	// Elements that are allowed to be empty
	selfClosing := map[string]bool{
		"img": true, "br": true, "hr": true, "input": true,
		"meta": true, "link": true, "area": true, "base": true,
		"col": true, "embed": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}

	// Multiple passes since removing a child might make parent empty
	for i := 0; i < 3; i++ {
		removed := 0
		doc.Find("*").Each(func(_ int, s *goquery.Selection) {
			tagName := goquery.NodeName(s)

			// Skip self-closing elements
			if selfClosing[tagName] {
				return
			}

			// Skip if should keep
			if c.shouldKeep(s) {
				return
			}

			// Check if empty
			text := strings.TrimSpace(s.Text())
			children := s.Children().Length()

			if len(text) == 0 && children == 0 {
				result.Stats.EmptyElementRemovals++
				result.Stats.RecordRemoval(tagName)
				s.Remove()
				removed++
			}
		})
		if removed == 0 {
			break
		}
	}
}

// collapseWhitespace normalizes whitespace in text nodes.
func (c *Cleaner) collapseWhitespace(doc *goquery.Document, result *Result) {
	// This will be handled in the output phase
}

// whitespaceRegex matches multiple whitespace characters.
var whitespaceRegex = regexp.MustCompile(`\s+`)

// commentRegex matches HTML comments.
var commentRegex = regexp.MustCompile(`<!--[\s\S]*?-->`)

// generateOutput produces the final output in the configured format.
func (c *Cleaner) generateOutput(doc *goquery.Document, result *Result) (string, error) {
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

	// Convert to requested output format
	switch c.config.Output {
	case OutputText:
		return c.htmlToText(html), nil
	case OutputMarkdown:
		// For now, just return HTML. Full markdown conversion would need another library.
		result.AddWarning("output", "Markdown output not yet implemented, returning HTML", "")
		return html, nil
	default:
		return html, nil
	}
}

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
