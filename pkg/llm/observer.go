// Package llm provides a unified interface for LLM providers.
package llm

import (
	"context"
	"time"
)

// LLMObserver receives notifications about LLM calls for observability.
// Implement this interface to integrate with any observability backend
// (Langfuse, Sentry, custom logging, metrics, etc.).
//
// The observer is called after every LLM call, whether successful or failed.
// This allows tracking of:
// - Token usage and costs
// - Latency metrics
// - Error rates and types
// - Model performance
type LLMObserver interface {
	// OnLLMCall is called after each LLM provider execution.
	// It receives full context about the call including request, response, and timing.
	//
	// Parameters:
	//   - ctx: The context from the original call (may contain trace IDs, user info, etc.)
	//   - event: Contains all details about the LLM call
	//
	// Implementations should be non-blocking or handle their own goroutines
	// to avoid impacting extraction latency.
	OnLLMCall(ctx context.Context, event LLMCallEvent)
}

// LLMCallEvent contains all information about an LLM call.
type LLMCallEvent struct {
	// Provider name (e.g., "anthropic", "openai", "openrouter")
	Provider string

	// Model used for the call (may differ from requested for auto-routing)
	Model string

	// Request details
	Request LLMCallRequest

	// Response details (nil if call failed before getting a response)
	Response *LLMCallResponse

	// Error if the call failed (nil on success)
	Error error

	// Duration of the LLM call
	Duration time.Duration

	// Attempt number (0 = first attempt, 1 = first retry, etc.)
	Attempt int

	// Timestamp when the call started
	StartedAt time.Time
}

// LLMCallRequest contains the request sent to the LLM.
type LLMCallRequest struct {
	// Messages sent to the LLM
	Messages []Message

	// MaxTokens requested
	MaxTokens int

	// Temperature used
	Temperature float64

	// Whether strict JSON mode was enabled
	StrictMode bool

	// Input content size in bytes (the webpage content being extracted)
	InputContentSize int
}

// LLMCallResponse contains the response from the LLM.
type LLMCallResponse struct {
	// Raw response content
	Content string

	// Token usage
	InputTokens  int
	OutputTokens int

	// Finish reason ("stop", "length", etc.)
	FinishReason string

	// Generation ID for cost tracking (provider-specific)
	GenerationID string

	// Cost if provider returns it inline
	Cost         float64
	CostIncluded bool
}

// ObserverFunc is a convenience type for using a function as an LLMObserver.
type ObserverFunc func(ctx context.Context, event LLMCallEvent)

// OnLLMCall implements LLMObserver.
func (f ObserverFunc) OnLLMCall(ctx context.Context, event LLMCallEvent) {
	f(ctx, event)
}

// MultiObserver combines multiple observers into one.
// All observers are called for each event.
type MultiObserver struct {
	observers []LLMObserver
}

// NewMultiObserver creates an observer that dispatches to multiple observers.
func NewMultiObserver(observers ...LLMObserver) *MultiObserver {
	return &MultiObserver{observers: observers}
}

// OnLLMCall dispatches the event to all registered observers.
func (m *MultiObserver) OnLLMCall(ctx context.Context, event LLMCallEvent) {
	for _, obs := range m.observers {
		obs.OnLLMCall(ctx, event)
	}
}

// Add adds an observer to the multi-observer.
func (m *MultiObserver) Add(obs LLMObserver) {
	m.observers = append(m.observers, obs)
}
