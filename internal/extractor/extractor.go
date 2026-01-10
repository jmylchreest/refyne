package extractor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/refyne/refyne/internal/llm"
	"github.com/refyne/refyne/internal/logger"
	"github.com/refyne/refyne/pkg/schema"
)

// ExtractionResult holds extracted and validated data.
type ExtractionResult struct {
	Data        any                      // Extracted structured data
	Raw         string                   // Raw LLM response
	Errors      []schema.ValidationError // Validation errors (if any)
	Usage       llm.Usage                // Token usage
	RetryCount  int                      // Number of retries performed
	LLMDuration time.Duration            // Time spent in LLM calls
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
	logger.Debug("extractor starting",
		"schema", s.Name,
		"content_size", len(content),
		"max_retries", e.config.MaxRetries)

	var lastErr error
	var totalUsage llm.Usage
	var totalLLMDuration time.Duration

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		logger.Debug("extractor attempt", "attempt", attempt+1, "max_attempts", e.config.MaxRetries+1)

		llmStart := time.Now()
		result, err := e.extractOnce(ctx, content, s, lastErr)
		llmDuration := time.Since(llmStart)
		totalLLMDuration += llmDuration

		// Accumulate token usage across attempts
		totalUsage.InputTokens += result.Usage.InputTokens
		totalUsage.OutputTokens += result.Usage.OutputTokens

		logger.Debug("extractor attempt complete",
			"attempt", attempt+1,
			"input_tokens", result.Usage.InputTokens,
			"output_tokens", result.Usage.OutputTokens,
			"error", err)

		if err == nil {
			// Validate the result
			validationErrors := s.Validate(result.Data)
			logger.Debug("extractor validation complete",
				"validation_errors", len(validationErrors))

			if len(validationErrors) == 0 {
				result.Usage = totalUsage
				result.RetryCount = attempt
				result.LLMDuration = totalLLMDuration
				logger.Debug("extractor success",
					"total_attempts", attempt+1,
					"total_input_tokens", totalUsage.InputTokens,
					"total_output_tokens", totalUsage.OutputTokens,
					"llm_duration", totalLLMDuration)
				return result, nil
			}

			// Validation failed - retry with error context
			lastErr = &validationErrorWrapper{errors: validationErrors}
			result.Errors = validationErrors
			logger.Debug("extractor validation failed, will retry",
				"errors", validationErrors)
		} else {
			lastErr = err
		}

		// Check if we should retry
		if attempt >= e.config.MaxRetries {
			logger.Debug("extractor max retries reached", "max_retries", e.config.MaxRetries)
			break
		}

		if !isRetryable(err) && lastErr != nil {
			// If we have validation errors, those are retryable
			if _, ok := lastErr.(*validationErrorWrapper); !ok {
				logger.Debug("extractor error not retryable", "error", lastErr)
				break
			}
		}
	}

	// Return last result with errors
	logger.Debug("extractor failed",
		"attempts", e.config.MaxRetries+1,
		"error", lastErr)
	return ExtractionResult{
		Usage:       totalUsage,
		RetryCount:  e.config.MaxRetries,
		LLMDuration: totalLLMDuration,
	}, fmt.Errorf("extraction failed after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// extractOnce performs a single extraction attempt.
func (e *Extractor) extractOnce(ctx context.Context, content string, s schema.Schema, previousErr error) (ExtractionResult, error) {
	logger.Debug("extractor building prompt",
		"has_previous_error", previousErr != nil)

	prompt := BuildExtractionPrompt(content, s, previousErr)
	logger.Debug("extractor prompt built", "prompt_size", len(prompt))

	jsonSchema, err := s.ToJSONSchema()
	if err != nil {
		logger.Debug("extractor JSON schema generation failed", "error", err)
		return ExtractionResult{}, fmt.Errorf("failed to generate JSON schema: %w", err)
	}
	logger.Debug("extractor JSON schema generated")

	logger.Debug("extractor calling LLM",
		"provider", e.provider.Name(),
		"max_tokens", e.config.MaxTokens,
		"temperature", e.config.Temperature)

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
		logger.Debug("extractor LLM completion failed", "error", err)
		return ExtractionResult{Usage: llm.Usage{}}, fmt.Errorf("LLM completion failed: %w", err)
	}
	logger.Debug("extractor LLM response received",
		"response_size", len(resp.Content),
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens)

	// Parse response
	data, err := s.Unmarshal([]byte(resp.Content))
	if err != nil {
		logger.Debug("extractor failed to parse response", "error", err)
		return ExtractionResult{
			Raw:   resp.Content,
			Usage: resp.Usage,
		}, fmt.Errorf("failed to parse response as JSON: %w (response: %s)", err, truncateForError(resp.Content))
	}

	logger.Debug("extractor response parsed successfully")
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
