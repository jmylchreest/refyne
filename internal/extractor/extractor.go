package extractor

import (
	"context"
	"fmt"
	"strings"

	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/pkg/schema"
)

// ExtractionResult holds extracted and validated data.
type ExtractionResult struct {
	Data       any                      // Extracted structured data
	Raw        string                   // Raw LLM response
	Errors     []schema.ValidationError // Validation errors (if any)
	Usage      llm.Usage                // Token usage
	RetryCount int                      // Number of retries performed
}

// Extractor performs LLM-based data extraction.
type Extractor struct {
	provider llm.Provider
	config   Config
}

// Config holds extractor settings.
type Config struct {
	MaxRetries  int
	Temperature float64
	MaxTokens   int
	DebugMode   bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxRetries:  3,
		Temperature: 0.1,
		MaxTokens:   8192,
	}
}

// New creates a new Extractor.
func New(provider llm.Provider, opts ...Option) *Extractor {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Extractor{
		provider: provider,
		config:   cfg,
	}
}

// Option configures the extractor.
type Option func(*Config)

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(n int) Option {
	return func(c *Config) {
		c.MaxRetries = n
	}
}

// WithTemperature sets the LLM temperature.
func WithTemperature(t float64) Option {
	return func(c *Config) {
		c.Temperature = t
	}
}

// WithMaxTokens sets the maximum tokens for responses.
func WithMaxTokens(n int) Option {
	return func(c *Config) {
		c.MaxTokens = n
	}
}

// WithDebugMode enables debug output.
func WithDebugMode(enabled bool) Option {
	return func(c *Config) {
		c.DebugMode = enabled
	}
}

// Extract performs LLM-based data extraction from content.
func (e *Extractor) Extract(ctx context.Context, content string, s schema.Schema) (ExtractionResult, error) {
	var lastErr error
	var totalUsage llm.Usage

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		result, err := e.extractOnce(ctx, content, s, lastErr)

		// Accumulate token usage across attempts
		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens

		if err == nil {
			// Validate the result
			validationErrors := s.Validate(result.Data)
			if len(validationErrors) == 0 {
				result.Usage = totalUsage
				result.RetryCount = attempt
				return result, nil
			}

			// Validation failed - retry with error context
			lastErr = &validationErrorWrapper{errors: validationErrors}
			result.Errors = validationErrors
		} else {
			lastErr = err
		}

		// Check if we should retry
		if attempt >= e.config.MaxRetries {
			break
		}

		if !isRetryable(err) && lastErr != nil {
			// If we have validation errors, those are retryable
			if _, ok := lastErr.(*validationErrorWrapper); !ok {
				break
			}
		}
	}

	// Return last result with errors
	return ExtractionResult{
		Usage:      totalUsage,
		RetryCount: e.config.MaxRetries,
	}, fmt.Errorf("extraction failed after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// extractOnce performs a single extraction attempt.
func (e *Extractor) extractOnce(ctx context.Context, content string, s schema.Schema, previousErr error) (ExtractionResult, error) {
	prompt := BuildExtractionPrompt(content, s, previousErr)

	jsonSchema, err := s.ToJSONSchema()
	if err != nil {
		return ExtractionResult{}, fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	resp, err := e.provider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: GetSystemPrompt()},
			{Role: llm.RoleUser, Content: prompt},
		},
		MaxTokens:   e.config.MaxTokens,
		Temperature: e.config.Temperature,
		JSONSchema:  jsonSchema,
	})
	if err != nil {
		return ExtractionResult{Usage: llm.Usage{}}, fmt.Errorf("LLM completion failed: %w", err)
	}

	// Parse response
	data, err := s.Unmarshal([]byte(resp.Content))
	if err != nil {
		return ExtractionResult{
			Raw:   resp.Content,
			Usage: resp.Usage,
		}, fmt.Errorf("failed to parse response as JSON: %w (response: %s)", err, truncateForError(resp.Content))
	}

	return ExtractionResult{
		Data:  data,
		Raw:   resp.Content,
		Usage: resp.Usage,
	}, nil
}

// validationErrorWrapper wraps validation errors for retry context.
type validationErrorWrapper struct {
	errors []schema.ValidationError
}

func (e *validationErrorWrapper) Error() string {
	var sb strings.Builder
	for _, err := range e.errors {
		sb.WriteString("- Field \"")
		sb.WriteString(err.Field)
		sb.WriteString("\": ")
		sb.WriteString(err.Message)
		sb.WriteString("\n")
	}
	return sb.String()
}

// isRetryable determines if an error should trigger a retry.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Validation errors are always retryable
	if _, ok := err.(*validationErrorWrapper); ok {
		return true
	}

	// Parse errors are retryable
	errStr := err.Error()
	if strings.Contains(errStr, "failed to parse") {
		return true
	}

	// Rate limit errors should be retried (would need proper error types)
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		return true
	}

	return false
}

// truncateForError truncates content for error messages.
func truncateForError(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "..."
}
