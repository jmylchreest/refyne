// Package openai provides an OpenAI-based extractor for structured data extraction.
package openai

import (
	"os"

	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/pkg/extractor"
)

// DefaultModel is the default OpenAI model.
const DefaultModel = "gpt-4o"

// Extractor performs extraction using OpenAI's API.
type Extractor struct {
	*extractor.BaseLLMExtractor
	available bool
}

// New creates a new OpenAI extractor.
// If cfg is nil, default configuration is used.
// API key is read from cfg.APIKey or OPENAI_API_KEY environment variable.
func New(cfg *extractor.LLMConfig) (*Extractor, error) {
	if cfg == nil {
		cfg = &extractor.LLMConfig{}
	}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	provider, err := llm.NewOpenAIProvider(llm.ProviderConfig{
		APIKey:  apiKey,
		BaseURL: cfg.BaseURL,
		Model:   model,
	})
	if err != nil {
		return &Extractor{available: false}, err
	}

	return &Extractor{
		BaseLLMExtractor: extractor.NewBaseLLMExtractor("openai", provider, cfg),
		available:        apiKey != "",
	}, nil
}

// Available returns true if the OpenAI API key is configured.
func (e *Extractor) Available() bool {
	return e.available
}
