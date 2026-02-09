package llm

import (
	"fmt"
	"os"
)

// ProviderFactory creates providers from config.
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

// DefaultModels maps provider names to their default models.
var DefaultModels = map[string]string{
	"anthropic":  "claude-sonnet-4-20250514",
	"openai":     "gpt-4o",
	"openrouter": "openrouter/auto",
	"ollama":     "llama3.2",
	"helicone":   "gpt-5-nano",
}

var registry = map[string]ProviderFactory{}

func init() {
	// Register all built-in providers
	RegisterProvider("anthropic", func(cfg ProviderConfig) (Provider, error) {
		return NewAnthropicProvider(cfg)
	})
	RegisterProvider("openai", func(cfg ProviderConfig) (Provider, error) {
		return NewOpenAIProvider(cfg)
	})
	RegisterProvider("openrouter", func(cfg ProviderConfig) (Provider, error) {
		return NewOpenRouterProvider(cfg)
	})
	RegisterProvider("ollama", func(cfg ProviderConfig) (Provider, error) {
		return NewOllamaProvider(cfg)
	})
	RegisterProvider("helicone", func(cfg ProviderConfig) (Provider, error) {
		return NewHeliconeProvider(cfg)
	})
}

// NewProvider creates a provider by name.
func NewProvider(name string, cfg ProviderConfig) (Provider, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: anthropic, openai, openrouter, ollama, helicone)", name)
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

// IsRegistered returns true if a provider is registered.
func IsRegistered(name string) bool {
	_, ok := registry[name]
	return ok
}

// providerEnvKeys maps provider names to their API key environment variables.
var providerEnvKeys = map[string]string{
	"openrouter": "OPENROUTER_API_KEY",
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"cerebras":   "CEREBRAS_API_KEY",
	"helicone":   "HELICONE_API_KEY",
}

// HasAPIKey checks if an API key environment variable is set for the given provider.
func HasAPIKey(provider string) bool {
	if envKey, ok := providerEnvKeys[provider]; ok {
		return os.Getenv(envKey) != ""
	}
	return false
}
