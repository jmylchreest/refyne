package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const (
	openRouterBaseURL       = "https://openrouter.ai/api/v1"
	openRouterModelsURL     = "https://openrouter.ai/api/v1/models"
	openRouterGenerationURL = "https://openrouter.ai/api/v1/generation"
)

// OpenRouterProvider implements the full Provider interface plus optional
// interfaces for model listing, cost tracking, and cost estimation.
type OpenRouterProvider struct {
	client      openai.Client
	httpClient  *http.Client
	model       string
	cfg         ProviderConfig
	apiKey      string
	httpReferer string
	appTitle    string

	// Model cache
	modelCache     map[string]*ModelInfo
	modelCacheMu   sync.RWMutex
	modelCacheTime time.Time
	cacheTTL       time.Duration
}

// NewOpenRouterProvider creates a new OpenRouter provider.
func NewOpenRouterProvider(cfg ProviderConfig) (*OpenRouterProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenRouter API key required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = openRouterBaseURL
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(baseURL),
	}

	// Add OpenRouter-specific headers
	if cfg.HTTPReferer != "" {
		opts = append(opts, option.WithHeader("HTTP-Referer", cfg.HTTPReferer))
	}
	if cfg.AppTitle != "" {
		opts = append(opts, option.WithHeader("X-Title", cfg.AppTitle))
	}

	client := openai.NewClient(opts...)

	model := cfg.Model
	if model == "" {
		model = "openrouter/auto"
	}

	return &OpenRouterProvider{
		client:      client,
		httpClient:  &http.Client{Timeout: 30 * time.Second}, // For metadata APIs
		model:       model,
		cfg:         cfg,
		apiKey:      cfg.APIKey,
		httpReferer: cfg.HTTPReferer,
		appTitle:    cfg.AppTitle,
		modelCache:  make(map[string]*ModelInfo),
		cacheTTL:    1 * time.Hour,
	}, nil
}

// Execute sends a completion request to OpenRouter.
func (p *OpenRouterProvider) Execute(ctx context.Context, req Request) (*Response, error) {
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
		return nil, fmt.Errorf("OpenRouter API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Response{
		Content:      resp.Choices[0].Message.Content,
		FinishReason: string(resp.Choices[0].FinishReason),
		Usage: Usage{
			InputTokens:  int(resp.Usage.PromptTokens),
			OutputTokens: int(resp.Usage.CompletionTokens),
		},
		Model:        resp.Model,
		GenerationID: resp.ID,
		Duration:     time.Since(start),
	}, nil
}

// Name returns the provider identifier.
func (p *OpenRouterProvider) Name() string {
	return "openrouter"
}

// Model returns the configured model name.
func (p *OpenRouterProvider) Model() string {
	return p.model
}

// ListModels fetches all available models from OpenRouter.
// Results are cached for the configured TTL.
func (p *OpenRouterProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if err := p.ensureModelCache(ctx); err != nil {
		return nil, err
	}

	p.modelCacheMu.RLock()
	defer p.modelCacheMu.RUnlock()

	models := make([]ModelInfo, 0, len(p.modelCache))
	for _, m := range p.modelCache {
		models = append(models, *m)
	}
	return models, nil
}

// GetModelInfo returns metadata for a specific model.
func (p *OpenRouterProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	if err := p.ensureModelCache(ctx); err != nil {
		return nil, err
	}

	p.modelCacheMu.RLock()
	defer p.modelCacheMu.RUnlock()

	if info, ok := p.modelCache[modelID]; ok {
		return info, nil
	}
	return nil, nil
}

// SupportsGenerationCost returns true - OpenRouter supports generation cost lookup.
func (p *OpenRouterProvider) SupportsGenerationCost() bool {
	return true
}

// GetGenerationCost fetches the actual cost from OpenRouter for a generation.
func (p *OpenRouterProvider) GetGenerationCost(ctx context.Context, generationID string) (float64, error) {
	if generationID == "" {
		return 0, fmt.Errorf("generation ID required")
	}

	// Retry with delays - generation stats may not be immediately available
	retryDelays := []time.Duration{200 * time.Millisecond, 500 * time.Millisecond, 1000 * time.Millisecond}

	var lastErr error
	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelays[attempt-1])
		}

		cost, err := p.fetchGenerationCost(ctx, generationID)
		if err == nil {
			return cost, nil
		}
		lastErr = err

		// Only retry on 404 (not found yet) - other errors are not recoverable
		if !strings.Contains(err.Error(), "status 404") {
			return 0, err
		}
	}

	return 0, lastErr
}

func (p *OpenRouterProvider) fetchGenerationCost(ctx context.Context, generationID string) (float64, error) {
	url := fmt.Sprintf("%s?id=%s", openRouterGenerationURL, generationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			TotalCost float64 `json:"total_cost"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data.TotalCost, nil
}

// EstimateCost calculates estimated cost based on cached pricing.
func (p *OpenRouterProvider) EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (float64, error) {
	info, err := p.GetModelInfo(ctx, modelID)
	if err != nil {
		return 0, err
	}

	if info == nil {
		// Model not found - return fallback estimate
		return p.estimateCostFallback(modelID, inputTokens, outputTokens), nil
	}

	inputCost := float64(inputTokens) * info.PromptPrice
	outputCost := float64(outputTokens) * info.CompletionPrice
	return inputCost + outputCost, nil
}

// estimateCostFallback provides rough cost estimates when pricing data isn't available.
func (p *OpenRouterProvider) estimateCostFallback(model string, inputTokens, outputTokens int) float64 {
	// Per million tokens
	inputPricePer1M := 0.25
	outputPricePer1M := 1.00

	modelLower := strings.ToLower(model)
	switch {
	case strings.Contains(modelLower, "gpt-4") && !strings.Contains(modelLower, "mini"):
		inputPricePer1M = 15.0
		outputPricePer1M = 60.0
	case strings.Contains(modelLower, "claude-3-opus"), strings.Contains(modelLower, "claude-opus"):
		inputPricePer1M = 15.0
		outputPricePer1M = 75.0
	case strings.Contains(modelLower, "gpt-3.5"), strings.Contains(modelLower, "claude-3-sonnet"), strings.Contains(modelLower, "claude-3.5"):
		inputPricePer1M = 3.0
		outputPricePer1M = 15.0
	case strings.Contains(modelLower, "gpt-4o-mini"), strings.Contains(modelLower, "claude-3-haiku"):
		inputPricePer1M = 0.15
		outputPricePer1M = 0.60
	case strings.Contains(modelLower, "llama"), strings.Contains(modelLower, "mixtral"), strings.Contains(modelLower, "gemma"):
		inputPricePer1M = 0.10
		outputPricePer1M = 0.40
	case strings.Contains(modelLower, ":free"):
		return 0
	}

	inputCost := float64(inputTokens) * inputPricePer1M / 1_000_000
	outputCost := float64(outputTokens) * outputPricePer1M / 1_000_000
	return inputCost + outputCost
}

// ensureModelCache refreshes the model cache if stale.
func (p *OpenRouterProvider) ensureModelCache(ctx context.Context) error {
	p.modelCacheMu.RLock()
	isStale := time.Since(p.modelCacheTime) > p.cacheTTL
	p.modelCacheMu.RUnlock()

	if !isStale {
		return nil
	}

	return p.refreshModelCache(ctx)
}

func (p *OpenRouterProvider) refreshModelCache(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterModelsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result openRouterModelsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse and cache
	cache := make(map[string]*ModelInfo, len(result.Data))
	for _, m := range result.Data {
		promptPrice := parsePrice(m.Pricing.Prompt)
		completionPrice := parsePrice(m.Pricing.Completion)

		info := &ModelInfo{
			ID:              m.ID,
			Name:            m.Name,
			Description:     m.Description,
			ContextLength:   m.ContextLength,
			PromptPrice:     promptPrice,
			CompletionPrice: completionPrice,
			IsFree:          promptPrice == 0 && completionPrice == 0,
			Capabilities:    parseOpenRouterCapabilities(m.SupportedParameters),
		}

		// Extract max_completion_tokens from top_provider if available
		if m.TopProvider != nil && m.TopProvider.MaxCompletionTokens > 0 {
			info.MaxCompletionTokens = m.TopProvider.MaxCompletionTokens
		}

		cache[m.ID] = info
	}

	p.modelCacheMu.Lock()
	p.modelCache = cache
	p.modelCacheTime = time.Now()
	p.modelCacheMu.Unlock()

	return nil
}

// OpenRouter API response types

type openRouterModelsResponse struct {
	Data []openRouterModel `json:"data"`
}

type openRouterModel struct {
	ID                  string                   `json:"id"`
	Name                string                   `json:"name"`
	Description         string                   `json:"description"`
	Pricing             openRouterPricing        `json:"pricing"`
	ContextLength       int                      `json:"context_length"`
	SupportedParameters []string                 `json:"supported_parameters"`
	TopProvider         *openRouterTopProvider   `json:"top_provider,omitempty"`
}

type openRouterTopProvider struct {
	ContextLength       int  `json:"context_length,omitempty"`
	MaxCompletionTokens int  `json:"max_completion_tokens,omitempty"`
	IsModerated         bool `json:"is_moderated,omitempty"`
}

type openRouterPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Image      string `json:"image,omitempty"`
	Request    string `json:"request,omitempty"`
}

func parsePrice(s string) float64 {
	if s == "" || s == "0" {
		return 0
	}
	var price float64
	_, _ = fmt.Sscanf(s, "%f", &price)
	return price
}

func parseOpenRouterCapabilities(params []string) ModelCapabilities {
	caps := ModelCapabilities{
		SupportsStreaming: true, // OpenRouter always supports streaming
	}

	for _, p := range params {
		switch p {
		case "structured_outputs":
			caps.SupportsStructuredOutputs = true
		case "tools", "tool_choice":
			caps.SupportsTools = true
		case "reasoning":
			caps.SupportsReasoning = true
		case "response_format":
			caps.SupportsResponseFormat = true
		case "vision":
			caps.SupportsVision = true
		}
	}

	return caps
}

// Ensure OpenRouterProvider implements all interfaces
var (
	_ Provider          = (*OpenRouterProvider)(nil)
	_ ModelLister       = (*OpenRouterProvider)(nil)
	_ ModelInfoProvider = (*OpenRouterProvider)(nil)
	_ CostTracker       = (*OpenRouterProvider)(nil)
	_ CostEstimator     = (*OpenRouterProvider)(nil)
)
