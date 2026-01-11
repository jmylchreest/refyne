// Package openrouter provides an OpenRouter-based extractor for structured data extraction.
package openrouter

import (
	"os"

	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/pkg/extractor"
)

// DefaultModel is the default OpenRouter model.
// "openrouter/auto" uses OpenRouter's automatic model selection.
const DefaultModel = "openrouter/auto"

// DefaultBaseURL is the OpenRouter API endpoint.
const DefaultBaseURL = "https://openrouter.ai/api/v1"

// Extractor performs extraction using OpenRouter's API.
type Extractor struct {
	*extractor.BaseLLMExtractor
	available bool
}

// New creates a new OpenRouter extractor.
// If cfg is nil, default configuration is used.
// API key is read from cfg.APIKey or OPENROUTER_API_KEY environment variable.
func New(cfg *extractor.LLMConfig) (*Extractor, error) {
	if cfg == nil {
		cfg = &extractor.LLMConfig{}
	}

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	provider, err := llm.NewOpenRouterProvider(llm.ProviderConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return &Extractor{available: false}, err
	}

	return &Extractor{
		BaseLLMExtractor: extractor.NewBaseLLMExtractor("openrouter", provider, cfg),
		available:        apiKey != "",
	}, nil
}

// Available returns true if the OpenRouter API key is configured.
func (e *Extractor) Available() bool {
	return e.available
}
