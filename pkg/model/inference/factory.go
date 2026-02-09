package inference

import (
	"fmt"

	"github.com/jmylchreest/refyne/pkg/llm"
)

// DetectProvider auto-detects the best available inference provider
// based on environment variables and returns a configured Inferencer.
//
// Priority:
//  1. OPENROUTER_API_KEY → OpenRouter
//  2. ANTHROPIC_API_KEY → Anthropic
//  3. OPENAI_API_KEY → OpenAI
//  4. Ollama (localhost, no key)
func DetectProvider() (Inferencer, error) {
	name, key := llm.DetectProvider()
	return NewRemote(name, WithAPIKey(key))
}

// New creates an Inferencer from a provider name and options.
// For remote providers, use names like "openrouter", "anthropic", "openai", "ollama", "helicone".
// For local models, use "local" with a model path.
func New(provider string, opts ...RemoteOption) (Inferencer, error) {
	if provider == "local" {
		// Extract model path from options — need a different approach
		return nil, fmt.Errorf("use NewLocal() for local inference")
	}
	return NewRemote(provider, opts...)
}

// AvailableProviders returns the names of all registered remote providers
// plus "local" for GGUF support.
func AvailableProviders() []string {
	providers := llm.AvailableProviders()
	return append(providers, "local")
}

// DetectProviderName returns the provider name and API key
// without creating an Inferencer. Useful for configuration display.
func DetectProviderName() (name string, apiKey string) {
	return llm.DetectProvider()
}

// HasAPIKey checks if an API key is set for the given provider.
// It delegates to the llm package's provider registry to avoid
// duplicating env var name mappings.
func HasAPIKey(provider string) bool {
	return llm.HasAPIKey(provider)
}
