package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Known Anthropic model pricing (per token, USD)
// Updated periodically - these are fallback values
var anthropicPricing = map[string]struct {
	promptPrice     float64
	completionPrice float64
	contextLength   int
}{
	"claude-opus-4-20250514":          {15.0 / 1_000_000, 75.0 / 1_000_000, 200000},
	"claude-sonnet-4-20250514":        {3.0 / 1_000_000, 15.0 / 1_000_000, 200000},
	"claude-3-5-sonnet-20241022":      {3.0 / 1_000_000, 15.0 / 1_000_000, 200000},
	"claude-3-5-haiku-20241022":       {0.80 / 1_000_000, 4.0 / 1_000_000, 200000},
	"claude-3-opus-20240229":          {15.0 / 1_000_000, 75.0 / 1_000_000, 200000},
	"claude-3-sonnet-20240229":        {3.0 / 1_000_000, 15.0 / 1_000_000, 200000},
	"claude-3-haiku-20240307":         {0.25 / 1_000_000, 1.25 / 1_000_000, 200000},
}

// AnthropicProvider implements Provider with ModelLister and CostEstimator.
// Anthropic doesn't support per-generation cost tracking (uses org-level Admin API instead).
type AnthropicProvider struct {
	client anthropic.Client
	model  string
	cfg    ProviderConfig
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(cfg ProviderConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic API key required")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
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

// Execute sends a completion request to Anthropic.
func (p *AnthropicProvider) Execute(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

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
	// Anthropic doesn't support response_format, uses tool_use instead
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
		return nil, fmt.Errorf("anthropic API error: %w", err)
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
				return nil, fmt.Errorf("failed to marshal tool input: %w", err)
			}
			content = string(jsonBytes)
		}
	}

	usage := Usage{
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
	}

	// Calculate cost inline since we have the usage
	cost, _ := p.EstimateCost(ctx, p.model, usage.InputTokens, usage.OutputTokens)

	return &Response{
		Content:      content,
		FinishReason: string(resp.StopReason),
		Usage:        usage,
		Model:        string(resp.Model),
		Cost:         cost,
		CostIncluded: true, // We calculated from known pricing
		Duration:     time.Since(start),
	}, nil
}

// Name returns the provider identifier.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Model returns the configured model name.
func (p *AnthropicProvider) Model() string {
	return p.model
}

// ListModels returns available Anthropic models.
// Note: Anthropic has a models API but the SDK may not expose it directly.
// For now, return known models with their capabilities.
func (p *AnthropicProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Return known models - Anthropic's model list is relatively static
	models := []ModelInfo{
		{
			ID:              "claude-opus-4-20250514",
			Name:            "Claude Opus 4",
			Description:     "Most capable model for complex tasks",
			ContextLength:   200000,
			PromptPrice:     15.0 / 1_000_000,
			CompletionPrice: 75.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true, // Via tool_use
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsReasoning:         true,
				SupportsVision:            true,
			},
		},
		{
			ID:              "claude-sonnet-4-20250514",
			Name:            "Claude Sonnet 4",
			Description:     "Best balance of speed and capability",
			ContextLength:   200000,
			PromptPrice:     3.0 / 1_000_000,
			CompletionPrice: 15.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsReasoning:         true,
				SupportsVision:            true,
			},
		},
		{
			ID:              "claude-3-5-sonnet-20241022",
			Name:            "Claude 3.5 Sonnet",
			Description:     "Previous generation Sonnet",
			ContextLength:   200000,
			PromptPrice:     3.0 / 1_000_000,
			CompletionPrice: 15.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsVision:            true,
			},
		},
		{
			ID:              "claude-3-5-haiku-20241022",
			Name:            "Claude 3.5 Haiku",
			Description:     "Fast and affordable",
			ContextLength:   200000,
			PromptPrice:     0.80 / 1_000_000,
			CompletionPrice: 4.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsVision:            true,
			},
		},
	}

	return models, nil
}

// GetModelInfo returns info for a specific Anthropic model.
func (p *AnthropicProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	for _, m := range models {
		if m.ID == modelID {
			return &m, nil
		}
	}
	return nil, nil
}

// EstimateCost calculates cost based on known Anthropic pricing.
func (p *AnthropicProvider) EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (float64, error) {
	if pricing, ok := anthropicPricing[modelID]; ok {
		return float64(inputTokens)*pricing.promptPrice + float64(outputTokens)*pricing.completionPrice, nil
	}

	// Fallback to Sonnet pricing for unknown models
	return float64(inputTokens)*(3.0/1_000_000) + float64(outputTokens)*(15.0/1_000_000), nil
}

// Ensure AnthropicProvider implements required interfaces
var (
	_ Provider          = (*AnthropicProvider)(nil)
	_ ModelLister       = (*AnthropicProvider)(nil)
	_ ModelInfoProvider = (*AnthropicProvider)(nil)
	_ CostEstimator     = (*AnthropicProvider)(nil)
	// Note: Anthropic does NOT implement CostTracker - no per-generation cost lookup
)
