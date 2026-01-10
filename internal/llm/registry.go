package llm

import (
	"fmt"
)

// ProviderFactory creates providers.
type ProviderFactory func(cfg ProviderConfig) (Provider, error)

var registry = map[string]ProviderFactory{
	"anthropic": func(cfg ProviderConfig) (Provider, error) {
		return NewAnthropicProvider(cfg)
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
