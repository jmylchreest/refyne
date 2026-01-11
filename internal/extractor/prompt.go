// Package extractor handles LLM-based data extraction from web content.
package extractor

import (
	"strings"

	"github.com/refyne/refyne/internal/logger"
	"github.com/refyne/refyne/pkg/schema"
)

const systemPrompt = `You are a data extraction assistant. Your task is to extract structured data from webpage content.

Rules:
1. Extract only the data that matches the schema fields
2. Return valid JSON matching the exact schema structure
3. If a required field cannot be found, use null
4. If an optional field cannot be found, omit it
5. For URLs, use absolute URLs when possible
6. For prices/numbers, extract the numeric value only (no currency symbols)
7. Be precise and extract exactly what is requested`

// BuildExtractionPrompt creates the prompt for LLM extraction.
func BuildExtractionPrompt(content string, s schema.Schema, previousErr error, maxContentSize int) string {
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
	prompt.WriteString(truncateContent(content, maxContentSize))
	prompt.WriteString("\n```\n")

	return prompt.String()
}

// GetSystemPrompt returns the system prompt for extraction.
func GetSystemPrompt() string {
	return systemPrompt
}

// truncateContent limits content size to avoid token limits.
// maxLen of 0 means no limit.
func truncateContent(content string, maxLen int) string {
	if maxLen <= 0 || len(content) <= maxLen {
		return content
	}
	logger.Warn("content truncated due to length",
		"original_bytes", len(content),
		"max_bytes", maxLen,
		"truncated_bytes", len(content)-maxLen)
	return content[:maxLen] + "\n\n[Content truncated due to length...]"
}
