// Package inference provides a unified interface for model inference,
// supporting both local GGUF models and remote API providers.
//
// This package replaces pkg/llm as the core model execution abstraction.
// The existing pkg/llm providers are re-exported as remote backends here.
package inference

import (
	"context"
	"time"
)

// Inferencer runs model inference (prompt in, text out).
// This is the low-level layer that loads a model and runs prompts.
type Inferencer interface {
	// Infer sends a prompt and returns the model's response.
	Infer(ctx context.Context, req Request) (*Response, error)

	// Name returns the inferencer identifier (e.g., "openrouter", "local-gguf").
	Name() string

	// Available reports whether this inferencer is ready to use.
	Available() bool

	// Close releases any resources held by the inferencer.
	Close() error
}

// Request represents an inference request.
type Request struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Grammar     string         // Optional GBNF grammar for constrained decoding (local only)
	JSONSchema  map[string]any // For structured output (remote providers)
	StrictMode  bool           // Strict JSON schema validation
}

// Message represents a chat message.
type Message struct {
	Role         Role
	Content      string
	CacheControl string // "ephemeral" for prompt caching on supported providers
}

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Response represents the result of an inference call.
type Response struct {
	Content      string
	Usage        Usage
	Model        string
	Provider     string
	Duration     time.Duration
	FinishReason string  // "stop" or "length"
	GenerationID string  // Provider-specific ID for cost tracking
	Cost         float64 // Actual cost if available
	CostIncluded bool    // True if Cost contains actual cost
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
}
