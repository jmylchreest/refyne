// Package refyne provides a highly configurable HTML content cleaner.
// It uses heuristic-based approaches to reduce token usage while preserving
// meaningful content for LLM extraction.
package refyne

// OutputFormat specifies the output format of the cleaner.
type OutputFormat string

const (
	OutputHTML     OutputFormat = "html"
	OutputText     OutputFormat = "text"
	OutputMarkdown OutputFormat = "markdown"
)

// Config defines all configuration options for the refyne cleaner.
type Config struct {
	// === Removal Options ===

	// StripScripts removes <script> tags and their contents.
	StripScripts bool `json:"strip_scripts"`

	// StripStyles removes <style> tags and style="" attributes.
	StripStyles bool `json:"strip_styles"`

	// StripComments removes HTML comments.
	StripComments bool `json:"strip_comments"`

	// StripHiddenElements removes elements with display:none, visibility:hidden,
	// or the hidden attribute.
	StripHiddenElements bool `json:"strip_hidden_elements"`

	// StripEmptyElements removes elements with no text content.
	StripEmptyElements bool `json:"strip_empty_elements"`

	// StripEventHandlers removes onclick, onload, and other event attributes.
	StripEventHandlers bool `json:"strip_event_handlers"`

	// StripSVGContent removes inline SVG elements (often decorative icons).
	StripSVGContent bool `json:"strip_svg_content"`

	// StripIframes removes iframe elements.
	StripIframes bool `json:"strip_iframes"`

	// StripNoscript removes noscript fallback content.
	StripNoscript bool `json:"strip_noscript"`

	// UnwrapNoscript replaces <noscript> tags with their contents.
	// This is useful because markdown converters typically ignore noscript content.
	// Only applies when StripNoscript is false.
	UnwrapNoscript bool `json:"unwrap_noscript"`

	// === Attribute Cleaning ===

	// StripDataAttributes removes data-* attributes.
	StripDataAttributes bool `json:"strip_data_attributes"`

	// StripClasses removes class="" attributes.
	StripClasses bool `json:"strip_classes"`

	// StripIDs removes id="" attributes.
	StripIDs bool `json:"strip_ids"`

	// StripARIA removes aria-* attributes.
	StripARIA bool `json:"strip_aria"`

	// StripMicrodata removes itemscope, itemprop, itemtype attributes.
	StripMicrodata bool `json:"strip_microdata"`

	// === Whitespace ===

	// CollapseWhitespace normalizes runs of whitespace to single spaces.
	CollapseWhitespace bool `json:"collapse_whitespace"`

	// TrimElements trims leading/trailing whitespace from element text.
	TrimElements bool `json:"trim_elements"`

	// === Preservation (overrides removals) ===

	// PreserveLinks keeps <a> elements with href attributes.
	PreserveLinks bool `json:"preserve_links"`

	// PreserveImages keeps <img> elements.
	PreserveImages bool `json:"preserve_images"`

	// PreserveTables keeps table structure (table, tr, td, th, thead, tbody).
	PreserveTables bool `json:"preserve_tables"`

	// PreserveForms keeps form elements (form, input, select, textarea, label, button).
	PreserveForms bool `json:"preserve_forms"`

	// PreserveLists keeps list structure (ul, ol, li, dl, dt, dd).
	PreserveLists bool `json:"preserve_lists"`

	// PreserveSemanticTags keeps semantic HTML5 elements
	// (article, main, section, aside, nav, header, footer, figure, figcaption).
	PreserveSemanticTags bool `json:"preserve_semantic_tags"`

	// === Selector-based Rules ===

	// RemoveSelectors is a list of CSS selectors to always remove.
	RemoveSelectors []string `json:"remove_selectors"`

	// KeepSelectors is a list of CSS selectors to always keep (overrides removals).
	KeepSelectors []string `json:"keep_selectors"`

	// === Heuristics ===

	// RemoveByLinkDensity removes elements where link text / total text > threshold.
	RemoveByLinkDensity bool `json:"remove_by_link_density"`

	// LinkDensityThreshold is the ratio above which an element is considered navigation.
	// Default: 0.5 (50% of text is links).
	LinkDensityThreshold float64 `json:"link_density_threshold"`

	// RemoveShortText removes elements with less than MinTextLength characters.
	RemoveShortText bool `json:"remove_short_text"`

	// MinTextLength is the minimum text characters to keep an element.
	MinTextLength int `json:"min_text_length"`

	// === Output ===

	// Output specifies the output format: html, text, or markdown.
	Output OutputFormat `json:"output"`

	// Debug enables verbose logging of what was removed and why.
	Debug bool `json:"debug"`
}

// DefaultConfig returns a sensible default configuration that works for most content.
// It aggressively removes scripts, styles, and hidden elements while preserving
// all meaningful content structures.
func DefaultConfig() *Config {
	return &Config{
		// Safe removals - these never contain useful extraction content
		StripScripts:        true,
		StripStyles:         true,
		StripComments:       true,
		StripEventHandlers:  true,
		StripNoscript:       false, // Keep noscript - often contains image fallbacks for JS-loaded content
		UnwrapNoscript:      true,  // Unwrap noscript content so markdown converters can process it
		StripSVGContent:     true,
		StripIframes:        true,
		StripHiddenElements: true,

		// Attribute cleaning - reduce noise while keeping structure
		StripDataAttributes: true,
		StripARIA:           true,
		StripMicrodata:      false, // Sometimes useful for structured data
		StripClasses:        false, // Selectors may need them
		StripIDs:            false, // Anchors may need them

		// Conservative - don't remove these by default
		StripEmptyElements: false, // Risky - might remove meaningful spacers

		// Preserve all meaningful content
		PreserveLinks:        true,
		PreserveImages:       true,
		PreserveTables:       true,
		PreserveForms:        true,
		PreserveLists:        true,
		PreserveSemanticTags: true,

		// Heuristics off by default - too aggressive for general use
		RemoveByLinkDensity:  false,
		LinkDensityThreshold: 0.5,
		RemoveShortText:      false,
		MinTextLength:        20,

		// Common ad/tracking selectors that are safe to remove
		RemoveSelectors: []string{
			// Hidden elements
			"[aria-hidden='true']",
			"[hidden]",

			// Google AdSense / DoubleClick
			"ins.adsbygoogle",
			"[data-ad-client]",
			"[data-ad-slot]",
			"[data-google-query-id]",
			"iframe[src*='doubleclick']",
			"iframe[src*='googlesyndication']",

			// Common ad classes/IDs
			".ad",
			".ads",
			".advert",
			".advertisement",
			".ad-container",
			".ad-wrapper",
			".ad-slot",
			".ad-unit",
			".ad-banner",
			"#ad",
			"#ads",
			"[class*='sponsored']",
			"[id*='google_ads']",

			// Social sharing buttons (not content)
			".share-buttons",
			".social-share",
			".sharing-buttons",

			// Cookie banners / GDPR notices
			"[class*='cookie']",
			"[id*='cookie']",
			"[class*='gdpr']",
			"[class*='consent']",

			// Newsletter popups
			"[class*='newsletter']",
			"[class*='subscribe']",
			".popup",
			".modal-backdrop",
		},

		// Whitespace normalization
		CollapseWhitespace: true,
		TrimElements:       true,

		// Default to HTML output
		Output: OutputHTML,
		Debug:  false,
	}
}

// PresetMinimal returns a minimal cleaning config that only removes
// scripts, styles, and comments. Use when you want maximum content preservation.
func PresetMinimal() *Config {
	return &Config{
		StripScripts:       true,
		StripStyles:        true,
		StripComments:      true,
		CollapseWhitespace: true,
		Output:             OutputHTML,
	}
}

// PresetAggressive returns an aggressive cleaning config for articles
// and blog posts. Enables link density heuristics and removes navigation.
func PresetAggressive() *Config {
	cfg := DefaultConfig()
	cfg.RemoveByLinkDensity = true
	cfg.LinkDensityThreshold = 0.5
	cfg.RemoveShortText = true
	cfg.MinTextLength = 25
	cfg.StripEmptyElements = true
	cfg.RemoveSelectors = append(cfg.RemoveSelectors,
		"nav",
		"header",
		"footer",
		"aside",
		".sidebar",
		".navigation",
		".nav",
		".menu",
		".ad",
		".ads",
		".advertisement",
		".banner",
		".cookie",
		".popup",
		".modal",
		"[role='navigation']",
		"[role='banner']",
		"[role='contentinfo']",
	)
	return cfg
}

// Merge merges another config into this one.
// Non-zero/non-empty values from other override this config.
// Selectors are appended, not replaced.
func (c *Config) Merge(other *Config) *Config {
	if other == nil {
		return c
	}

	// Create a copy
	merged := *c

	// Merge removal options (other wins if true)
	if other.StripScripts {
		merged.StripScripts = true
	}
	if other.StripStyles {
		merged.StripStyles = true
	}
	if other.StripComments {
		merged.StripComments = true
	}
	if other.StripHiddenElements {
		merged.StripHiddenElements = true
	}
	if other.StripEmptyElements {
		merged.StripEmptyElements = true
	}
	if other.StripEventHandlers {
		merged.StripEventHandlers = true
	}
	if other.StripSVGContent {
		merged.StripSVGContent = true
	}
	if other.StripIframes {
		merged.StripIframes = true
	}
	if other.StripNoscript {
		merged.StripNoscript = true
	}
	if other.UnwrapNoscript {
		merged.UnwrapNoscript = true
	}

	// Merge attribute options
	if other.StripDataAttributes {
		merged.StripDataAttributes = true
	}
	if other.StripClasses {
		merged.StripClasses = true
	}
	if other.StripIDs {
		merged.StripIDs = true
	}
	if other.StripARIA {
		merged.StripARIA = true
	}
	if other.StripMicrodata {
		merged.StripMicrodata = true
	}

	// Merge heuristics
	if other.RemoveByLinkDensity {
		merged.RemoveByLinkDensity = true
	}
	if other.LinkDensityThreshold > 0 {
		merged.LinkDensityThreshold = other.LinkDensityThreshold
	}
	if other.RemoveShortText {
		merged.RemoveShortText = true
	}
	if other.MinTextLength > 0 {
		merged.MinTextLength = other.MinTextLength
	}

	// Append selectors (deduplicated)
	if len(other.RemoveSelectors) > 0 {
		seen := make(map[string]bool)
		for _, s := range merged.RemoveSelectors {
			seen[s] = true
		}
		for _, s := range other.RemoveSelectors {
			if !seen[s] {
				merged.RemoveSelectors = append(merged.RemoveSelectors, s)
				seen[s] = true
			}
		}
	}
	if len(other.KeepSelectors) > 0 {
		seen := make(map[string]bool)
		for _, s := range merged.KeepSelectors {
			seen[s] = true
		}
		for _, s := range other.KeepSelectors {
			if !seen[s] {
				merged.KeepSelectors = append(merged.KeepSelectors, s)
				seen[s] = true
			}
		}
	}

	return &merged
}
