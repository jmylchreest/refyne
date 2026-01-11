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

// BaseLLMExtractor provides common extraction logic for LLM-based extractors.
// Provider-specific extractors embed this and provide their own LLM provider.
type BaseLLMExtractor struct {
	provider llm.Provider
	config   LLMConfig
	name     string
}

// NewBaseLLMExtractor creates a new base extractor with the given provider.
// This is used internally by provider-specific extractors (anthropic, openai, etc.).
func NewBaseLLMExtractor(name string, provider llm.Provider, cfg *LLMConfig) *BaseLLMExtractor {
	config := DefaultLLMConfig()
	if cfg != nil {
		if cfg.Temperature > 0 {
			config.Temperature = cfg.Temperature
		}
		if cfg.MaxTokens > 0 {
			config.MaxTokens = cfg.MaxTokens
		}
		if cfg.MaxRetries > 0 {
			config.MaxRetries = cfg.MaxRetries
		}
		if cfg.MaxContentSize > 0 {
			config.MaxContentSize = cfg.MaxContentSize
		}
	}

	return &BaseLLMExtractor{
		provider: provider,
		config:   config,
		name:     name,
	}
}

// Extract performs LLM-based data extraction with retry logic.
func (e *BaseLLMExtractor) Extract(ctx context.Context, content string, s schema.Schema) (*Result, error) {
	logger.Debug("extractor starting",
		"extractor", e.name,
		"schema", s.Name,
		"content_size", len(content),
		"max_retries", e.config.MaxRetries)

	var lastErr error
	var totalUsage Usage
	var totalDuration time.Duration

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		logger.Debug("extractor attempt", "attempt", attempt+1, "max_attempts", e.config.MaxRetries+1)

		start := time.Now()
		result, err := e.extractOnce(ctx, content, s, lastErr)
		duration := time.Since(start)
		totalDuration += duration

		// Accumulate token usage
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
			logger.Debug("extractor validation complete", "validation_errors", len(validationErrors))

			if len(validationErrors) == 0 {
				result.Usage = totalUsage
				result.RetryCount = attempt
				result.Duration = totalDuration
				logger.Debug("extractor success",
					"total_attempts", attempt+1,
					"total_input_tokens", totalUsage.InputTokens,
					"total_output_tokens", totalUsage.OutputTokens,
					"duration", totalDuration,
					"model", result.Model)
				return result, nil
			}

			// Validation failed - retry with error context
			lastErr = &validationError{errors: validationErrors}
			result.Errors = validationErrors
			logger.Debug("extractor validation failed, will retry", "errors", validationErrors)
		} else {
			lastErr = err
		}

		// Check if we should retry
		if attempt >= e.config.MaxRetries {
			logger.Debug("extractor max retries reached", "max_retries", e.config.MaxRetries)
			break
		}

		if !isRetryable(err) && lastErr != nil {
			if _, ok := lastErr.(*validationError); !ok {
				logger.Debug("extractor error not retryable", "error", lastErr)
				break
			}
		}
	}

	logger.Debug("extractor failed", "attempts", e.config.MaxRetries+1, "error", lastErr)
	return &Result{
		Usage:      totalUsage,
		RetryCount: e.config.MaxRetries,
		Duration:   totalDuration,
		Provider:   e.name,
	}, fmt.Errorf("extraction failed after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// extractOnce performs a single extraction attempt.
func (e *BaseLLMExtractor) extractOnce(ctx context.Context, content string, s schema.Schema, previousErr error) (*Result, error) {
	logger.Debug("extractor building prompt",
		"has_previous_error", previousErr != nil,
		"max_content_size", e.config.MaxContentSize)

	prompt := BuildPrompt(content, s, previousErr, e.config.MaxContentSize)
	logger.Debug("extractor prompt built", "prompt_size", len(prompt))

	jsonSchema, err := s.ToJSONSchema()
	if err != nil {
		logger.Debug("extractor JSON schema generation failed", "error", err)
		return &Result{}, fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	logger.Debug("extractor calling LLM",
		"provider", e.provider.Name(),
		"model", e.provider.Model(),
		"max_tokens", e.config.MaxTokens,
		"temperature", e.config.Temperature)

	resp, err := e.provider.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: SystemPrompt},
			{Role: llm.RoleUser, Content: prompt},
		},
		MaxTokens:   e.config.MaxTokens,
		Temperature: e.config.Temperature,
		JSONSchema:  jsonSchema,
	})
	if err != nil {
		logger.Debug("extractor LLM completion failed", "error", err)
		return &Result{
			Usage: Usage{},
		}, fmt.Errorf("LLM completion failed: %w", err)
	}

	logger.Debug("extractor LLM response received",
		"response_size", len(resp.Content),
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens)

	// Strip markdown code blocks if present
	jsonContent := StripMarkdownCodeBlock(resp.Content)

	// Parse response
	data, err := s.Unmarshal([]byte(jsonContent))
	if err != nil {
		logger.Debug("extractor failed to parse response", "error", err)
		return &Result{
			Raw: resp.Content,
			Usage: Usage{
				InputTokens:  resp.Usage.InputTokens,
				OutputTokens: resp.Usage.OutputTokens,
			},
		}, fmt.Errorf("failed to parse response as JSON: %w (response: %s)", err, truncateForError(resp.Content))
	}

	logger.Debug("extractor response parsed successfully")
	return &Result{
		Data:       data,
		Raw:        resp.Content,
		RawContent: content,
		Usage: Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
		Model:    resp.Model,
		Provider: e.name,
	}, nil
}

// Name returns the extractor name.
func (e *BaseLLMExtractor) Name() string {
	return e.name
}

// validationError wraps validation errors for retry context.
type validationError struct {
	errors []schema.ValidationError
}

func (e *validationError) Error() string {
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

	if _, ok := err.(*validationError); ok {
		return true
	}

	errStr := err.Error()
	if strings.Contains(errStr, "failed to parse") {
		return true
	}
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
