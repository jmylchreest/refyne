package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider wraps the Anthropic SDK.
type AnthropicProvider struct {
	client anthropic.Client
	model  string
	cfg    ProviderConfig
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg ProviderConfig) (*AnthropicProvider, error) {
	opts := []option.RequestOption{}

	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	if cfg.MaxRetries > 0 {
		opts = append(opts, option.WithMaxRetries(cfg.MaxRetries))
	}

	client := anthropic.NewClient(opts...)

	model := cfg.Model
	if model == "" {
		model = string(anthropic.ModelClaudeSonnet4_20250514)
	}

	return &AnthropicProvider{
		client: client,
		model:  model,
		cfg:    cfg,
	}, nil
}

// Complete sends a completion request to Anthropic.
func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	messages := make([]anthropic.MessageParam, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleSystem:
			systemPrompt = msg.Content
		case RoleUser:
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case RoleAssistant:
			messages = append(messages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.model),
		MaxTokens: int64(maxTokens),
		Messages:  messages,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	// Use tool-based JSON extraction for reliable structured output
	if req.JSONSchema != nil {
		properties, _ := req.JSONSchema["properties"].(map[string]any)
		required, _ := req.JSONSchema["required"].([]any)

		requiredStrings := make([]string, 0, len(required))
		for _, r := range required {
			if s, ok := r.(string); ok {
				requiredStrings = append(requiredStrings, s)
			}
		}

		params.Tools = []anthropic.ToolUnionParam{
			{
				OfTool: &anthropic.ToolParam{
					Name:        "extract_data",
					Description: anthropic.String("Extract structured data from the content"),
					InputSchema: anthropic.ToolInputSchemaParam{
						Type:       "object",
						Properties: properties,
						Required:   requiredStrings,
					},
				},
			},
		}
		params.ToolChoice = anthropic.ToolChoiceParamOfTool("extract_data")
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("anthropic API error: %w", err)
	}

	// Extract content from response
	var content string
	for _, block := range resp.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			content = b.Text
		case anthropic.ToolUseBlock:
			// For structured output, the tool input IS the extracted data
			jsonBytes, err := json.Marshal(b.Input)
			if err != nil {
				return CompletionResponse{}, fmt.Errorf("failed to marshal tool input: %w", err)
			}
			content = string(jsonBytes)
		}
	}

	return CompletionResponse{
		Content:      content,
		FinishReason: string(resp.StopReason),
		Usage: Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
		},
		Model: string(resp.Model),
	}, nil
}

// Name returns the provider identifier.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// SupportsJSONSchema returns true as Anthropic supports tool-based structured output.
func (p *AnthropicProvider) SupportsJSONSchema() bool {
	return true
}
