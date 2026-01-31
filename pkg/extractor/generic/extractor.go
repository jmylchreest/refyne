// Package generic provides a registry-driven LLM extractor that works with any registered provider.
// This eliminates the need for separate extractor packages per provider.
package generic

import (
	"fmt"
	"os"
	"strings"

	"github.com/jmylchreest/refyne/pkg/extractor"
	"github.com/jmylchreest/refyne/pkg/llm"
)

// envKeyMap maps provider names to their environment variable names.
var envKeyMap = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"openrouter": "OPENROUTER_API_KEY",
	"ollama":     "", // No key required
	"helicone":   "HELICONE_API_KEY",
}

// Extractor performs extraction using any registered LLM provider.
type Extractor struct {
	*extractor.BaseLLMExtractor
	available bool
}

// New creates a new generic extractor for the specified provider.
// The provider must be registered in the llm registry.
// API key is read from cfg.APIKey or the provider's environment variable.
func New(providerName string, cfg *extractor.LLMConfig) (*Extractor, error) {
	if cfg == nil {
		cfg = &extractor.LLMConfig{}
	}

	// Resolve API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		if envVar, ok := envKeyMap[providerName]; ok && envVar != "" {
			apiKey = os.Getenv(envVar)
		}
	}

	// Get default model if not specified
	model := cfg.Model
	if model == "" {
		model = llm.GetDefaultModel(providerName)
	}

	// Check if provider is registered
	if !llm.IsRegistered(providerName) {
		return nil, fmt.Errorf("unknown provider: %s (available: %s)",
			providerName, strings.Join(llm.AvailableProviders(), ", "))
	}

	// Create provider via registry
	provider, err := llm.NewProvider(providerName, llm.ProviderConfig{
		APIKey:         apiKey,
		BaseURL:        cfg.BaseURL,
		Model:          model,
		TargetProvider: cfg.TargetProvider,
		TargetAPIKey:   cfg.TargetAPIKey,
	})
	if err != nil {
		return &Extractor{available: false}, err
	}

	// Determine if extractor is available (has required credentials)
	available := true
	if envVar, ok := envKeyMap[providerName]; ok && envVar != "" {
		available = apiKey != ""
	}

	return &Extractor{
		BaseLLMExtractor: extractor.NewBaseLLMExtractor(providerName, provider, cfg),
		available:        available,
	}, nil
}

// Available returns true if the provider's API key is configured (if required).
func (e *Extractor) Available() bool {
	return e.available
}
