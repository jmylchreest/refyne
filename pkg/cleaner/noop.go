package cleaner

// NoopCleaner passes content through without modification.
// Use this when the fetcher already returns clean content,
// or when you want the LLM to process raw HTML.
type NoopCleaner struct{}

// NewNoop creates a new no-op cleaner.
func NewNoop() *NoopCleaner {
	return &NoopCleaner{}
}

// Clean returns the input unchanged.
func (c *NoopCleaner) Clean(html string) (string, error) {
	return html, nil
}

// Name returns the cleaner type.
func (c *NoopCleaner) Name() string {
	return "noop"
}
