// Package ollama provides an Ollama-based extractor for structured data extraction.
package ollama

import (
	"net/http"
	"time"

	"github.com/jmylchreest/refyne/pkg/llm"
	"github.com/jmylchreest/refyne/pkg/extractor"
)

// DefaultModel is the default Ollama model.
const DefaultModel = "llama3.2"

// DefaultBaseURL is the default Ollama API endpoint.
const DefaultBaseURL = "http://localhost:11434"

// Extractor performs extraction using a local Ollama instance.
type Extractor struct {
	*extractor.BaseLLMExtractor
	baseURL string
}

// New creates a new Ollama extractor.
// If cfg is nil, default configuration is used.
func New(cfg *extractor.LLMConfig) (*Extractor, error) {
	if cfg == nil {
		cfg = &extractor.LLMConfig{}
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	provider, err := llm.NewOllamaProvider(llm.ProviderConfig{
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return nil, err
	}

	return &Extractor{
		BaseLLMExtractor: extractor.NewBaseLLMExtractor("ollama", provider, cfg),
		baseURL:          baseURL,
	}, nil
}

// Available returns true if Ollama is running and accessible.
func (e *Extractor) Available() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(e.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
