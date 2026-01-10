package llm

import (
	"fmt"
	"os"
)

// ProviderFactory creates providers.
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

// DefaultModels maps provider names to their default models.
var DefaultModels = map[string]string{
	"anthropic":  "claude-opus-4-5-20251101",
	"openai":     "gpt-4o",
	"openrouter": "xiaomi/mimo-v2-flash:free",
	"ollama":     "llama3.2",
}

var registry = map[string]ProviderFactory{
	"anthropic": func(cfg ProviderConfig) (Provider, error) {
		return NewAnthropicProvider(cfg)
	},
	"openai": func(cfg ProviderConfig) (Provider, error) {
		return NewOpenAIProvider(cfg)
	},
	"openrouter": func(cfg ProviderConfig) (Provider, error) {
		// OpenRouter uses OpenAI-compatible API
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://openrouter.ai/api/v1"
		}
		return NewOpenAIProvider(cfg)
	},
	"ollama": func(cfg ProviderConfig) (Provider, error) {
		return NewOllamaProvider(cfg)
	},
}

// NewProvider creates a provider by name.
func NewProvider(name string, cfg ProviderConfig) (Provider, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: anthropic, openai, openrouter, ollama)", name)
	}
	return factory(cfg)
}

// RegisterProvider adds a custom provider factory.
func RegisterProvider(name string, factory ProviderFactory) {
	registry[name] = factory
}

// AvailableProviders returns the list of registered providers.
func AvailableProviders() []string {
	providers := make([]string, 0, len(registry))
	for name := range registry {
		providers = append(providers, name)
	}
	return providers
}

// DetectProvider auto-detects the best provider based on available API keys.
// Returns the provider name and API key.
// Priority: OPENROUTER_API_KEY > ANTHROPIC_API_KEY > OPENAI_API_KEY > ollama (no key needed)
func DetectProvider() (provider string, apiKey string) {
	// Check OpenRouter first (often has free models)
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		return "openrouter", key
	}

	// Check Anthropic
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return "anthropic", key
	}

	// Check OpenAI
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return "openai", key
	}

	// Fall back to Ollama (no key required)
	return "ollama", ""
}

// GetDefaultModel returns the default model for a provider.
func GetDefaultModel(provider string) string {
	if model, ok := DefaultModels[provider]; ok {
		return model
	}
	return ""
}
