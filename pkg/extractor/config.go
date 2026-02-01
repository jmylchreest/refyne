package extractor

import (
	"strings"

	"github.com/jmylchreest/refyne/pkg/llm"
	"github.com/jmylchreest/refyne/pkg/schema"
)

// LLMConfig holds shared configuration for LLM-based extractors.
type LLMConfig struct {
	// Model overrides the default model for this provider.
	Model string

	// APIKey for the provider. If empty, checks environment variable.
	APIKey string

	// BaseURL for custom API endpoints.
	BaseURL string

	// Temperature for LLM responses (default: 0.1).
	Temperature float64

	// MaxTokens for LLM responses (default: 16384).
	// This is the minimum that most modern LLMs support for output.
	MaxTokens int

	// MaxRetries for rate limit errors only (default: 1).
	// Other errors (JSON parse, validation) are not retried.
	MaxRetries int

	// MaxContentSize limits input content in bytes (default: 100000, 0 = unlimited).
	MaxContentSize int

	// StrictMode enables strict JSON schema validation in the API request.
	// Only supported by OpenAI models (gpt-4o, gpt-4o-mini) and some OpenRouter models.
	// When enabled, the model must return valid JSON exactly matching the schema.
	// Default: false (for broader model compatibility).
	StrictMode bool

	// TargetProvider is the underlying provider for proxy providers (e.g., Helicone self-hosted).
	// For example, "openai" or "anthropic" when routing through Helicone.
	TargetProvider string

	// TargetAPIKey is the underlying provider's API key for proxy providers.
	// Required when using Helicone in self-hosted mode.
	TargetAPIKey string

	// HTTPReferer sets the HTTP-Referer header for OpenRouter attribution.
	HTTPReferer string

	// AppTitle sets the X-Title header for OpenRouter attribution.
	AppTitle string

	// Observer receives notifications about LLM calls for observability.
	// Use this to integrate with Langfuse, Sentry, custom logging, etc.
	// The observer is called after every LLM call (success or failure).
	Observer llm.LLMObserver
}

// DefaultLLMConfig returns sensible defaults for LLM extraction.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Temperature:    0.1,
		MaxTokens:      16384, // 16k is the minimum most modern models support
		MaxRetries:     1,     // Only retry rate limits, not JSON/validation errors
		MaxContentSize: 100000, // ~100KB
	}
}

// SystemPrompt is the shared system prompt for all LLM extractors.
const SystemPrompt = `You are a data extraction assistant. Extract structured data from webpage content.

Content may be provided as Markdown, HTML, or plain text.

Respond with ONLY valid JSON matching the schema. No explanations.

Rules:
1. Required fields: use null if not found
2. Optional fields: omit if not found
3. URLs: use absolute URLs when possible
4. Numbers: extract numeric value only (no currency symbols)`

// BuildPrompt creates the extraction prompt from content and schema.
func BuildPrompt(content string, s schema.Schema, previousErr error, maxContentSize int) string {
	var prompt strings.Builder

	prompt.WriteString("Extract structured data from the following webpage content.\n\n")
	prompt.WriteString(s.ToPromptDescription())

	// Include previous errors for self-correction
	if previousErr != nil {
		prompt.WriteString("\n## Previous Attempt Errors\n")
		prompt.WriteString("The previous extraction attempt had these errors that need to be fixed:\n")
		prompt.WriteString(previousErr.Error())
		prompt.WriteString("\n\nPlease correct these errors in your response.\n")
	}

	prompt.WriteString("\n## Webpage Content\n")
	prompt.WriteString("```\n")
	prompt.WriteString(TruncateContent(content, maxContentSize))
	prompt.WriteString("\n```\n")

	return prompt.String()
}

// TruncateContent limits content size to avoid token limits.
// maxLen of 0 means no limit.
func TruncateContent(content string, maxLen int) string {
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n\n[Content truncated due to length...]"
}

// StripMarkdownCodeBlock removes markdown code block wrappers from JSON responses.
// Some models wrap their JSON output in ```json ... ``` blocks.
func StripMarkdownCodeBlock(s string) string {
	s = strings.TrimSpace(s)

	// Check for ```json or ``` prefix
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	} else {
		return s // No code block wrapper
	}

	// Remove trailing ```
	s = strings.TrimSuffix(s, "```")

	return strings.TrimSpace(s)
}
