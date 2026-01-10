// Package llm provides a unified interface for LLM providers.
package llm

import (
	"context"
	"time"
)

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a chat message.
type Message struct {
	Role    Role
	Content string
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// CompletionRequest represents a request to the LLM.
type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	JSONSchema  map[string]any // For structured output
}

// CompletionResponse represents the LLM response.
type CompletionResponse struct {
	Content      string
	FinishReason string
	Usage        Usage
}

// Provider is the core abstraction over LLM backends.
type Provider interface {
	// Complete sends a completion request and returns structured output.
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

	// Name returns the provider identifier.
	Name() string

	// SupportsJSONSchema returns true if provider has native JSON mode.
	SupportsJSONSchema() bool
}

// ProviderConfig holds common configuration for providers.
type ProviderConfig struct {
	APIKey     string
	BaseURL    string // For OpenRouter or custom endpoints
	Model      string
	MaxRetries int
	Timeout    time.Duration
}

// DefaultProviderConfig returns sensible defaults.
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		MaxRetries: 3,
		Timeout:    60 * time.Second,
	}
}
