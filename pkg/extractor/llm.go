package extractor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmylchreest/refyne/pkg/llm"
	"github.com/jmylchreest/refyne/pkg/model/inference"
	"github.com/jmylchreest/refyne/internal/logger"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// BaseLLMExtractor provides common extraction logic for LLM-based extractors.
// Provider-specific extractors embed this and provide their own LLM provider.
type BaseLLMExtractor struct {
	provider   llm.Provider
	inferencer inference.Inferencer // New: wraps provider via inference layer
	config     LLMConfig
	name       string
	observer   llm.LLMObserver
}

// NewFromInferencer creates a BaseLLMExtractor backed by any Inferencer.
// This is the preferred constructor for new code.
func NewFromInferencer(inf inference.Inferencer, opts ...ExtractorOption) *BaseLLMExtractor {
	config := DefaultLLMConfig()
	for _, opt := range opts {
		opt(&config)
	}

	e := &BaseLLMExtractor{
		inferencer: inf,
		config:     config,
		name:       inf.Name(),
		observer:   config.Observer,
	}

	// If the inferencer wraps a pkg/llm provider, extract it for observer support
	if remote, ok := inf.(*inference.Remote); ok {
		e.provider = remote.Provider()
	}

	return e
}

// ExtractorOption configures a BaseLLMExtractor created via NewFromInferencer.
type ExtractorOption func(*LLMConfig)

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) ExtractorOption {
	return func(c *LLMConfig) { c.MaxRetries = n }
}

// WithTemperature sets the sampling temperature.
func WithTemperature(t float64) ExtractorOption {
	return func(c *LLMConfig) { c.Temperature = t }
}

// WithMaxTokens sets the maximum output tokens.
func WithMaxTokens(n int) ExtractorOption {
	return func(c *LLMConfig) { c.MaxTokens = n }
}

// WithStrictMode enables strict JSON schema validation.
func WithStrictMode(strict bool) ExtractorOption {
	return func(c *LLMConfig) { c.StrictMode = strict }
}

// WithObserver sets the LLM observer for observability.
func WithObserver(obs llm.LLMObserver) ExtractorOption {
	return func(c *LLMConfig) { c.Observer = obs }
}

// NewBaseLLMExtractor creates a new base extractor with the given provider.
// This is used internally by provider-specific extractors (anthropic, openai, etc.).
//
// Deprecated: Use NewFromInferencer instead.
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
		// StrictMode is a bool, so always copy it (false is a valid setting)
		config.StrictMode = cfg.StrictMode
	}

	return &BaseLLMExtractor{
		provider: provider,
		config:   config,
		name:     name,
		observer: cfg.Observer,
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
	var lastResult *Result // Track the last result for FinishReason on failure
	var totalUsage Usage
	var totalDuration time.Duration

	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		logger.Debug("extractor attempt", "attempt", attempt+1, "max_attempts", e.config.MaxRetries+1)

		start := time.Now()
		result, err := e.extractOnceWithAttempt(ctx, content, s, lastErr, attempt)
		lastResult = result // Preserve for FinishReason access on failure
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
	// Preserve FinishReason from last attempt for truncation detection
	var finishReason string
	var model string
	var generationID string
	if lastResult != nil {
		finishReason = lastResult.FinishReason
		model = lastResult.Model
		generationID = lastResult.GenerationID
	}
	return &Result{
		Usage:        totalUsage,
		RetryCount:   e.config.MaxRetries,
		Duration:     totalDuration,
		Provider:     e.name,
		Model:        model,
		GenerationID: generationID,
		FinishReason: finishReason,
	}, fmt.Errorf("extraction failed after %d attempts: %w", e.config.MaxRetries+1, lastErr)
}

// execute dispatches to either the inference.Inferencer or the legacy llm.Provider.
// This keeps extractOnceWithAttempt agnostic to the backend.
func (e *BaseLLMExtractor) execute(ctx context.Context, messages []llm.Message, jsonSchema map[string]any) (*llm.Response, error) {
	if e.inferencer != nil && e.provider == nil {
		// Use the inference.Inferencer path (no legacy provider available)
		req := inference.Request{
			MaxTokens:   e.config.MaxTokens,
			Temperature: e.config.Temperature,
			JSONSchema:  jsonSchema,
			StrictMode:  e.config.StrictMode,
		}
		for _, msg := range messages {
			m := inference.Message{
				Role:    inference.Role(msg.Role),
				Content: msg.Content,
			}
			if msg.CacheControl != "" {
				m.CacheControl = string(msg.CacheControl)
			}
			req.Messages = append(req.Messages, m)
		}

		resp, err := e.inferencer.Infer(ctx, req)
		if err != nil {
			return nil, err
		}

		return &llm.Response{
			Content:      resp.Content,
			Model:        resp.Model,
			Usage:        llm.Usage{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens},
			Duration:     resp.Duration,
			FinishReason: resp.FinishReason,
			GenerationID: resp.GenerationID,
			Cost:         resp.Cost,
			CostIncluded: resp.CostIncluded,
		}, nil
	}

	// Legacy llm.Provider path
	return e.provider.Execute(ctx, llm.Request{
		Messages:    messages,
		MaxTokens:   e.config.MaxTokens,
		Temperature: e.config.Temperature,
		JSONSchema:  jsonSchema,
		StrictMode:  e.config.StrictMode,
	})
}

// providerName returns the name of the underlying backend for logging.
func (e *BaseLLMExtractor) providerName() string {
	if e.provider != nil {
		return e.provider.Name()
	}
	return e.name
}

// providerModel returns the model identifier of the underlying backend.
func (e *BaseLLMExtractor) providerModel() string {
	if e.provider != nil {
		return e.provider.Model()
	}
	return e.name
}

// extractOnceWithAttempt performs a single extraction attempt with attempt tracking for observer.
func (e *BaseLLMExtractor) extractOnceWithAttempt(ctx context.Context, content string, s schema.Schema, previousErr error, attempt int) (*Result, error) {
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

	// Build messages with optional prompt caching
	systemMsg := llm.Message{Role: llm.RoleSystem, Content: SystemPrompt}
	if e.config.EnablePromptCaching {
		systemMsg.CacheControl = llm.CacheControlEphemeral
	}
	messages := []llm.Message{
		systemMsg,
		{Role: llm.RoleUser, Content: prompt},
	}

	logger.Debug("extractor calling LLM",
		"provider", e.providerName(),
		"model", e.providerModel(),
		"max_tokens", e.config.MaxTokens,
		"temperature", e.config.Temperature,
		"strict_mode", e.config.StrictMode)

	startedAt := time.Now()

	resp, err := e.execute(ctx, messages, jsonSchema)

	duration := time.Since(startedAt)

	// Notify observer (handles both success and failure)
	if e.observer != nil {
		event := llm.LLMCallEvent{
			Provider:  e.name,
			Model:     e.providerModel(),
			Duration:  duration,
			Attempt:   attempt,
			StartedAt: startedAt,
			Request: llm.LLMCallRequest{
				Messages:         messages,
				MaxTokens:        e.config.MaxTokens,
				Temperature:      e.config.Temperature,
				StrictMode:       e.config.StrictMode,
				InputContentSize: len(content),
			},
		}

		if err != nil {
			event.Error = err
		}

		if resp != nil {
			event.Model = resp.Model
			event.Response = &llm.LLMCallResponse{
				Content:      resp.Content,
				InputTokens:  resp.Usage.InputTokens,
				OutputTokens: resp.Usage.OutputTokens,
				FinishReason: resp.FinishReason,
				GenerationID: resp.GenerationID,
				Cost:         resp.Cost,
				CostIncluded: resp.CostIncluded,
			}
		}

		e.observer.OnLLMCall(ctx, event)
	}

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
			Model:        resp.Model,
			Provider:     e.name,
			GenerationID: resp.GenerationID,
			FinishReason: resp.FinishReason,
			Cost:         resp.Cost,
			CostIncluded: resp.CostIncluded,
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
		Model:        resp.Model,
		Provider:     e.name,
		GenerationID: resp.GenerationID,
		FinishReason: resp.FinishReason,
		Cost:         resp.Cost,
		CostIncluded: resp.CostIncluded,
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
// Only transient errors (rate limits) are retryable.
// JSON parse errors and validation errors are NOT retried - if the model
// can't produce valid output, retrying wastes time and tokens.
// The caller should use a fallback chain for model failures instead.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Only retry rate limits - these are transient
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
		return true
	}

	// Don't retry:
	// - JSON parse errors (model can't produce valid JSON)
	// - Validation errors (model output doesn't match schema)
	// - Other errors (auth, network, etc.)
	return false
}

// truncateForError truncates content for error messages.
func truncateForError(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "..."
}
