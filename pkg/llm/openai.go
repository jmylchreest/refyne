package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Known OpenAI model pricing (per token, USD)
var openaiPricing = map[string]struct {
	promptPrice     float64
	completionPrice float64
	contextLength   int
}{
	"gpt-4o":            {2.50 / 1_000_000, 10.0 / 1_000_000, 128000},
	"gpt-4o-mini":       {0.15 / 1_000_000, 0.60 / 1_000_000, 128000},
	"gpt-4-turbo":       {10.0 / 1_000_000, 30.0 / 1_000_000, 128000},
	"gpt-4":             {30.0 / 1_000_000, 60.0 / 1_000_000, 8192},
	"gpt-3.5-turbo":     {0.50 / 1_000_000, 1.50 / 1_000_000, 16385},
	"o1":                {15.0 / 1_000_000, 60.0 / 1_000_000, 200000},
	"o1-mini":           {3.0 / 1_000_000, 12.0 / 1_000_000, 128000},
	"o1-preview":        {15.0 / 1_000_000, 60.0 / 1_000_000, 128000},
}

// OpenAIProvider implements Provider for direct OpenAI API access.
type OpenAIProvider struct {
	client openai.Client
	model  string
	cfg    ProviderConfig
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg ProviderConfig) (*OpenAIProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key required")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}

	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := openai.NewClient(opts...)

	model := cfg.Model
	if model == "" {
		model = string(openai.ChatModelGPT4o)
	}

	return &OpenAIProvider{
		client: client,
		model:  model,
		cfg:    cfg,
	}, nil
}

// Execute sends a completion request to OpenAI.
func (p *OpenAIProvider) Execute(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		switch msg.Role {
		case RoleSystem:
			messages = append(messages, openai.SystemMessage(msg.Content))
		case RoleUser:
			messages = append(messages, openai.UserMessage(msg.Content))
		case RoleAssistant:
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := openai.ChatCompletionNewParams{
		Model:       openai.ChatModel(p.model),
		Messages:    messages,
		MaxTokens:   openai.Int(int64(maxTokens)),
		Temperature: openai.Float(req.Temperature),
	}

	// Use native JSON mode / structured outputs if schema provided
	if req.JSONSchema != nil {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "extraction_result",
					Schema: req.JSONSchema,
					Strict: openai.Bool(req.StrictMode),
				},
			},
		}
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	usage := Usage{
		InputTokens:  int(resp.Usage.PromptTokens),
		OutputTokens: int(resp.Usage.CompletionTokens),
	}

	// Calculate cost inline
	cost, _ := p.EstimateCost(ctx, p.model, usage.InputTokens, usage.OutputTokens)

	return &Response{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		Usage:        usage,
		Model:        resp.Model,
		Cost:         cost,
		CostIncluded: true,
		Duration:     time.Since(start),
	}, nil
}

// Name returns the provider identifier.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Model returns the configured model name.
func (p *OpenAIProvider) Model() string {
	return p.model
}

// ListModels returns available OpenAI models.
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// Return known models with capabilities
	models := []ModelInfo{
		{
			ID:              "gpt-4o",
			Name:            "GPT-4o",
			Description:     "Most capable GPT-4 model",
			ContextLength:   128000,
			PromptPrice:     2.50 / 1_000_000,
			CompletionPrice: 10.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsResponseFormat:    true,
				SupportsVision:            true,
			},
		},
		{
			ID:              "gpt-4o-mini",
			Name:            "GPT-4o Mini",
			Description:     "Fast and affordable",
			ContextLength:   128000,
			PromptPrice:     0.15 / 1_000_000,
			CompletionPrice: 0.60 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsResponseFormat:    true,
				SupportsVision:            true,
			},
		},
		{
			ID:              "o1",
			Name:            "o1",
			Description:     "Reasoning model",
			ContextLength:   200000,
			PromptPrice:     15.0 / 1_000_000,
			CompletionPrice: 60.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsReasoning:         true,
			},
		},
		{
			ID:              "o1-mini",
			Name:            "o1 Mini",
			Description:     "Fast reasoning model",
			ContextLength:   128000,
			PromptPrice:     3.0 / 1_000_000,
			CompletionPrice: 12.0 / 1_000_000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsReasoning:         true,
			},
		},
	}

	return models, nil
}

// GetModelInfo returns info for a specific OpenAI model.
func (p *OpenAIProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
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

// EstimateCost calculates cost based on known OpenAI pricing.
func (p *OpenAIProvider) EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (float64, error) {
	// Try exact match first
	if pricing, ok := openaiPricing[modelID]; ok {
		return float64(inputTokens)*pricing.promptPrice + float64(outputTokens)*pricing.completionPrice, nil
	}

	// Try prefix matching for versioned models
	for id, pricing := range openaiPricing {
		if strings.HasPrefix(modelID, id) {
			return float64(inputTokens)*pricing.promptPrice + float64(outputTokens)*pricing.completionPrice, nil
		}
	}

	// Fallback to gpt-4o-mini pricing
	return float64(inputTokens)*(0.15/1_000_000) + float64(outputTokens)*(0.60/1_000_000), nil
}

// Ensure OpenAIProvider implements required interfaces
var (
	_ Provider          = (*OpenAIProvider)(nil)
	_ ModelLister       = (*OpenAIProvider)(nil)
	_ ModelInfoProvider = (*OpenAIProvider)(nil)
	_ CostEstimator     = (*OpenAIProvider)(nil)
	// Note: OpenAI does NOT implement CostTracker - no per-generation cost lookup API
)
