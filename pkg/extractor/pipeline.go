package extractor

import (
	"context"
	"strings"

	"github.com/jmylchreest/refyne/pkg/schema"
)

// PipelineExtractor runs extractors in sequence, passing results through.
// This is useful for chaining extraction steps (e.g., regex pre-extraction → LLM refinement).
type PipelineExtractor struct {
	extractors []Extractor
}

// NewPipeline creates a pipeline that runs extractors in sequence.
func NewPipeline(extractors ...Extractor) *PipelineExtractor {
	return &PipelineExtractor{
		extractors: extractors,
	}
}

// Extract runs each extractor in sequence.
// The final result is returned. Token usage and duration are accumulated.
func (p *PipelineExtractor) Extract(ctx context.Context, content string, s schema.Schema) (*Result, error) {
	var finalResult *Result
	var totalUsage Usage
	var totalRetries int

	for _, ext := range p.extractors {
		if !ext.Available() {
			continue
		}

		result, err := ext.Extract(ctx, content, s)
		if err != nil {
			return nil, err
		}

		// Accumulate usage
		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens
		totalRetries += result.RetryCount

		finalResult = result
	}

	if finalResult != nil {
		finalResult.Usage = totalUsage
		finalResult.RetryCount = totalRetries
		finalResult.Provider = p.Name()
	}

	return finalResult, nil
}

// Name returns the pipeline name.
func (p *PipelineExtractor) Name() string {
	var names []string
	for _, ext := range p.extractors {
		names = append(names, ext.Name())
	}
	return "pipeline(" + strings.Join(names, "->") + ")"
}

// Available returns true if all extractors in the pipeline are available.
func (p *PipelineExtractor) Available() bool {
	for _, ext := range p.extractors {
		if !ext.Available() {
			return false
		}
	}
	return len(p.extractors) > 0
}
