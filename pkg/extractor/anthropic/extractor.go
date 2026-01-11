// Package anthropic provides an Anthropic-based extractor for structured data extraction.
package anthropic

import (
	"os"

	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/pkg/extractor"
)

// DefaultModel is the default Anthropic model.
const DefaultModel = "claude-sonnet-4-20250514"

// Extractor performs extraction using Anthropic's API.
type Extractor struct {
	*extractor.BaseLLMExtractor
	available bool
}

// New creates a new Anthropic extractor.
// If cfg is nil, default configuration is used.
// API key is read from cfg.APIKey or ANTHROPIC_API_KEY environment variable.
func New(cfg *extractor.LLMConfig) (*Extractor, error) {
	if cfg == nil {
		cfg = &extractor.LLMConfig{}
	}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	provider, err := llm.NewAnthropicProvider(llm.ProviderConfig{
		APIKey: apiKey,
		Model:  model,
	})
	if err != nil {
		return &Extractor{available: false}, err
	}

	return &Extractor{
		BaseLLMExtractor: extractor.NewBaseLLMExtractor("anthropic", provider, cfg),
		available:        apiKey != "",
	}, nil
}

// Available returns true if the Anthropic API key is configured.
func (e *Extractor) Available() bool {
	return e.available
}
