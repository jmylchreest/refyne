package extractor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jmylchreest/refyne/pkg/schema"
)

// ErrNoExtractorAvailable is returned when no extractors in the fallback chain are available.
var ErrNoExtractorAvailable = errors.New("no extractor available")

// FallbackExtractor tries each extractor in order until one succeeds.
// This is useful for provider failover (e.g., try Anthropic, fall back to OpenAI).
type FallbackExtractor struct {
	extractors []Extractor
}

// NewFallback creates a fallback chain from the given extractors.
// Extractors are tried in order. Only available extractors are used.
func NewFallback(extractors ...Extractor) *FallbackExtractor {
	return &FallbackExtractor{
		extractors: extractors,
	}
}

// Extract tries each extractor in order until one succeeds.
func (f *FallbackExtractor) Extract(ctx context.Context, content string, s schema.Schema) (*Result, error) {
	var lastErr error
	var tried []string

	for _, ext := range f.extractors {
		if !ext.Available() {
			continue
		}

		tried = append(tried, ext.Name())
		result, err := ext.Extract(ctx, content, s)
		if err == nil {
			return result, nil
		}

		lastErr = err
		// Continue to next extractor
	}

	if len(tried) == 0 {
		return nil, ErrNoExtractorAvailable
	}

	return nil, fmt.Errorf("all extractors failed (tried: %s): %w", strings.Join(tried, ", "), lastErr)
}

// Name returns the fallback chain name.
func (f *FallbackExtractor) Name() string {
	var names []string
	for _, ext := range f.extractors {
		names = append(names, ext.Name())
	}
	return "fallback(" + strings.Join(names, "->") + ")"
}

// Available returns true if at least one extractor is available.
func (f *FallbackExtractor) Available() bool {
	for _, ext := range f.extractors {
		if ext.Available() {
			return true
		}
	}
	return false
}

// First returns the first available extractor, or nil if none available.
func (f *FallbackExtractor) First() Extractor {
	for _, ext := range f.extractors {
		if ext.Available() {
			return ext
		}
	}
	return nil
}
