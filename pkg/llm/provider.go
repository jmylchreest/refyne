// Package llm provides a unified interface for LLM providers.
//
// Deprecated: Use pkg/model/inference instead. This package is retained for
// backwards compatibility. The provider implementations here are wrapped by
// inference.Remote to provide the new Inferencer interface.
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

// CacheControlType specifies the cache control behavior for a message.
type CacheControlType string

const (
	// CacheControlEphemeral marks the message for caching with provider-default TTL.
	// Supported by: Anthropic, OpenRouter (Anthropic/Gemini models), Gemini
	CacheControlEphemeral CacheControlType = "ephemeral"
)

// Message represents a chat message.
type Message struct {
	Role    Role
	Content string
	// CacheControl enables prompt caching for this message on supported providers.
	// Set to CacheControlEphemeral to cache the message prefix.
	// Only effective on providers/models that support prompt caching.
	CacheControl CacheControlType
}

// Request represents a completion request to the LLM.
type Request struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	JSONSchema  map[string]any // For structured output
	StrictMode  bool           // Use strict JSON schema validation (only for supported models)
}

// Usage tracks token consumption.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Response represents the result of an LLM execution.
type Response struct {
	Content      string
	FinishReason string
	Usage        Usage
	Model        string  // Actual model used (may differ from requested for auto-routing)
	GenerationID string  // Provider-specific ID for cost tracking (e.g., OpenRouter generation ID)
	Cost         float64 // Actual cost if provider returns it inline (0 if not available)
	CostIncluded bool    // True if Cost field contains actual cost from provider
	Duration     time.Duration
	// CacheStats contains prompt caching information if available.
	CacheStats *CacheStats
}

// CacheStats contains information about prompt caching for a request.
type CacheStats struct {
	// CachedTokens is the number of input tokens that were served from cache.
	CachedTokens int
	// CacheDiscount is the cost savings from caching (negative = cache write cost).
	CacheDiscount float64
	// CacheHit indicates if the request hit the cache.
	CacheHit bool
}

// Provider is the core interface that all LLM backends must implement.
type Provider interface {
	// Execute sends a completion request and returns the response.
	// This is the only required method - all providers must implement it.
	Execute(ctx context.Context, req Request) (*Response, error)

	// Name returns the provider identifier (e.g., "openrouter", "anthropic").
	Name() string

	// Model returns the configured model name.
	Model() string
}

// ModelInfo contains metadata about a model including pricing and capabilities.
type ModelInfo struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Description         string            `json:"description,omitempty"`
	ContextLength       int               `json:"context_length"`
	MaxCompletionTokens int               `json:"max_completion_tokens,omitempty"` // Max output tokens (0 = unknown/unlimited)
	PromptPrice         float64           `json:"prompt_price"`                    // Price per token (USD)
	CompletionPrice     float64           `json:"completion_price"`                // Price per token (USD)
	ImagePrice          float64           `json:"image_price,omitempty"`
	IsFree              bool              `json:"is_free"`
	Capabilities        ModelCapabilities `json:"capabilities"`
}

// ModelCapabilities describes what features a model supports.
type ModelCapabilities struct {
	SupportsStructuredOutputs bool `json:"supports_structured_outputs"` // JSON schema enforcement
	SupportsTools             bool `json:"supports_tools"`              // Function/tool calling
	SupportsStreaming         bool `json:"supports_streaming"`
	SupportsReasoning         bool `json:"supports_reasoning"` // Extended thinking
	SupportsResponseFormat    bool `json:"supports_response_format"`
	SupportsVision            bool `json:"supports_vision"`
}

// ModelLister is an optional interface for providers that can list available models.
// Not all providers support this (e.g., some don't have a models API).
type ModelLister interface {
	// ListModels returns all available models with their metadata.
	// This may involve an API call; implementations should cache appropriately.
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// ModelInfoProvider is an optional interface for providers that can fetch
// info about a specific model without listing all models.
type ModelInfoProvider interface {
	// GetModelInfo returns metadata for a specific model.
	// Returns nil, nil if the model is not found.
	GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error)
}

// CostTracker is an optional interface for providers that support
// fetching actual costs after a request completes.
// Some providers (like OpenRouter) provide generation IDs that can be
// used to look up exact costs after the fact.
type CostTracker interface {
	// GetGenerationCost fetches the actual cost for a completed generation.
	// The generationID comes from Response.GenerationID.
	// Returns the cost in USD.
	GetGenerationCost(ctx context.Context, generationID string) (float64, error)

	// SupportsGenerationCost returns true if the provider supports cost lookup.
	// This allows callers to check before attempting a lookup.
	SupportsGenerationCost() bool
}

// CostEstimator is an optional interface for providers that can estimate
// costs based on token counts without making an API call.
type CostEstimator interface {
	// EstimateCost calculates an estimated cost based on token counts.
	// Uses cached pricing data when available.
	EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (float64, error)
}

// ProviderConfig holds common configuration for providers.
type ProviderConfig struct {
	APIKey     string
	BaseURL    string // For custom endpoints or OpenRouter
	Model      string
	MaxRetries int
	Timeout    time.Duration
	// HTTPReferer and Title for OpenRouter attribution
	HTTPReferer string
	AppTitle    string
	// For proxy providers (e.g., Helicone self-hosted)
	TargetProvider string // Underlying provider (e.g., "openai", "anthropic")
	TargetAPIKey   string // Underlying provider's API key
}

// DefaultProviderConfig returns sensible defaults.
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		MaxRetries: 3,
		Timeout:    120 * time.Second,
	}
}

// Helper functions to check provider capabilities

// CanListModels returns true if the provider implements ModelLister.
func CanListModels(p Provider) bool {
	_, ok := p.(ModelLister)
	return ok
}

// CanGetModelInfo returns true if the provider implements ModelInfoProvider.
func CanGetModelInfo(p Provider) bool {
	_, ok := p.(ModelInfoProvider)
	return ok
}

// CanTrackCost returns true if the provider implements CostTracker.
func CanTrackCost(p Provider) bool {
	_, ok := p.(CostTracker)
	return ok
}

// CanEstimateCost returns true if the provider implements CostEstimator.
func CanEstimateCost(p Provider) bool {
	_, ok := p.(CostEstimator)
	return ok
}

// AsModelLister returns the provider as a ModelLister if it implements the interface.
func AsModelLister(p Provider) (ModelLister, bool) {
	ml, ok := p.(ModelLister)
	return ml, ok
}

// AsCostTracker returns the provider as a CostTracker if it implements the interface.
func AsCostTracker(p Provider) (CostTracker, bool) {
	ct, ok := p.(CostTracker)
	return ct, ok
}

// AsCostEstimator returns the provider as a CostEstimator if it implements the interface.
func AsCostEstimator(p Provider) (CostEstimator, bool) {
	ce, ok := p.(CostEstimator)
	return ce, ok
}

