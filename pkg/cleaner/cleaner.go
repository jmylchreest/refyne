// Package cleaner provides interfaces and implementations for cleaning HTML content.
// Cleaners transform raw HTML into a format suitable for LLM extraction.
package cleaner

// Cleaner transforms HTML content into a cleaner format for extraction.
// The default implementation converts HTML to Markdown, preserving semantic structure.
type Cleaner interface {
	// Clean transforms the input HTML into a cleaned format.
	// The output format depends on the implementation (markdown, plain text, etc.).
	Clean(html string) (string, error)

	// Name returns the cleaner type for logging/debugging.
	Name() string
}
