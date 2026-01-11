// Package extractor provides the interface for structured data extraction.
package extractor

import (
	"context"
	"time"

	"github.com/refyne/refyne/pkg/schema"
)

// Extractor extracts structured data from content.
type Extractor interface {
	// Extract performs extraction from content using the provided schema.
	Extract(ctx context.Context, content string, s schema.Schema) (*Result, error)

	// Name returns the extractor identifier.
	Name() string

	// Available returns true if the extractor is properly configured
	// (e.g., has required API keys or services available).
	Available() bool
}

// Result holds the extraction output.
type Result struct {
	// Data is the extracted structured data.
	Data any

	// Raw is the raw response from the extractor (e.g., LLM response).
	Raw string

	// RawContent is the input content that was sent for extraction.
	// Useful for generating training data.
	RawContent string

	// Errors contains any validation errors from the extraction.
	Errors []schema.ValidationError

	// Usage tracks token consumption for LLM-based extractors.
	Usage Usage

	// Model is the actual model used (may differ from requested for auto-routing).
	Model string

	// Provider is the provider/extractor name.
	Provider string

	// RetryCount is the number of retries performed.
	RetryCount int

	// Duration is the total time spent extracting.
	Duration time.Duration
}

// Usage tracks token consumption for LLM-based extractors.
type Usage struct {
	InputTokens  int
	OutputTokens int
}
