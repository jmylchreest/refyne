package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIProvider wraps the OpenAI SDK.
type OpenAIProvider struct {
	client openai.Client
	model  string
	cfg    ProviderConfig
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg ProviderConfig) (*OpenAIProvider, error) {
	opts := []option.RequestOption{}

	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
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

// NewOpenRouterProvider creates an OpenAI-compatible client for OpenRouter.
func NewOpenRouterProvider(cfg ProviderConfig) (*OpenAIProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api/v1"
	}
	return NewOpenAIProvider(cfg)
}

// Complete sends a completion request to OpenAI.
func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
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
					Strict: openai.Bool(true),
				},
			},
		}
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("openai API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("no choices in response")
	}

	return CompletionResponse{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		Usage: Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
	}, nil
}

// Name returns the provider identifier.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// SupportsJSONSchema returns true as OpenAI supports structured outputs.
func (p *OpenAIProvider) SupportsJSONSchema() bool {
	return true
}

func init() {
	RegisterProvider("openai", func(cfg ProviderConfig) (Provider, error) {
		return NewOpenAIProvider(cfg)
	})
	RegisterProvider("openrouter", func(cfg ProviderConfig) (Provider, error) {
		return NewOpenRouterProvider(cfg)
	})
}
