package extractor

import (
	"strings"

	"github.com/refyne/refyne/pkg/schema"
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

	// MaxTokens for LLM responses (default: 8192).
	MaxTokens int

	// MaxRetries for failed extractions (default: 3).
	MaxRetries int

	// MaxContentSize limits input content in bytes (default: 100000, 0 = unlimited).
	MaxContentSize int
}

// DefaultLLMConfig returns sensible defaults for LLM extraction.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Temperature:    0.1,
		MaxTokens:      8192,
		MaxRetries:     3,
		MaxContentSize: 100000, // ~100KB
	}
}

// SystemPrompt is the shared system prompt for all LLM extractors.
const SystemPrompt = `You are a data extraction assistant. Your task is to extract structured data from webpage content.

CRITICAL: You MUST respond with ONLY valid JSON. Do not include any explanatory text, commentary, or markdown formatting. Your entire response must be parseable as JSON.

Rules:
1. Extract only the data that matches the schema fields
2. Return valid JSON matching the exact schema structure
3. If a required field cannot be found, use null
4. If an optional field cannot be found, omit it
5. For URLs, use absolute URLs when possible
6. For prices/numbers, extract the numeric value only (no currency symbols)
7. Be precise and extract exactly what is requested
8. NEVER explain your reasoning - just return the JSON object`

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

	prompt.WriteString("\nRespond with ONLY the JSON object. No explanations or markdown.\n")

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
