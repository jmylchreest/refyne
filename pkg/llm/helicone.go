package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// HeliconeCloudBaseURL is the default Helicone cloud gateway URL.
	HeliconeCloudBaseURL = "https://ai-gateway.helicone.ai"
	// HeliconeAPIBaseURL is the Helicone API URL for metadata/cost lookups.
	HeliconeAPIBaseURL = "https://api.helicone.ai"
	// HeliconeModelsURL is the public model registry endpoint (no auth required).
	HeliconeModelsURL = "https://api.helicone.ai/v1/public/model-registry/models"
)

// HeliconeProvider implements Provider for Helicone.
// Supports both cloud (managed keys) and self-hosted (proxy) modes.
type HeliconeProvider struct {
	cfg        ProviderConfig
	httpClient *http.Client
	selfHosted bool // True if using custom BaseURL (self-hosted mode)

	// Model cache for pricing/capabilities
	modelCache     map[string]*ModelInfo
	modelCacheMu   sync.RWMutex
	modelCacheTime time.Time
	cacheTTL       time.Duration
}

// NewHeliconeProvider creates a new Helicone provider.
func NewHeliconeProvider(cfg ProviderConfig) (*HeliconeProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("helicone API key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = HeliconeCloudBaseURL
	}

	// Detect self-hosted mode
	selfHosted := cfg.BaseURL != HeliconeCloudBaseURL

	// Self-hosted requires target provider key
	if selfHosted && cfg.TargetAPIKey == "" {
		return nil, fmt.Errorf("helicone self-hosted mode requires target provider API key (TargetAPIKey)")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-5-nano"
	}
	cfg.Model = model

	return &HeliconeProvider{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
		selfHosted: selfHosted,
		modelCache: make(map[string]*ModelInfo),
		cacheTTL:   1 * time.Hour,
	}, nil
}

// Name returns the provider identifier.
func (p *HeliconeProvider) Name() string {
	return "helicone"
}

// Model returns the configured model name.
func (p *HeliconeProvider) Model() string {
	return p.cfg.Model
}

// Execute sends a completion request through Helicone.
func (p *HeliconeProvider) Execute(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

	// Build endpoint URL
	endpoint := p.buildEndpoint()

	// Build request body (OpenAI-compatible format)
	body := p.buildRequestBody(req)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers based on mode
	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("helicone returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return p.parseResponse(respBody, time.Since(start))
}

func (p *HeliconeProvider) buildEndpoint() string {
	if p.selfHosted {
		// Self-hosted: /v1/gateway/{provider}/v1/chat/completions
		targetProvider := p.cfg.TargetProvider
		if targetProvider == "" {
			targetProvider = "oai" // Default to OpenAI
		}
		return fmt.Sprintf("%s/v1/gateway/%s/v1/chat/completions",
			strings.TrimSuffix(p.cfg.BaseURL, "/"), targetProvider)
	}
	// Cloud: /v1/chat/completions
	return fmt.Sprintf("%s/v1/chat/completions", strings.TrimSuffix(p.cfg.BaseURL, "/"))
}

func (p *HeliconeProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	if p.selfHosted {
		// Self-hosted: Helicone-Auth + provider Authorization
		req.Header.Set("Helicone-Auth", "Bearer "+p.cfg.APIKey)
		req.Header.Set("Authorization", "Bearer "+p.cfg.TargetAPIKey)
	} else {
		// Cloud: Standard Bearer auth (Helicone manages provider keys)
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
}

func (p *HeliconeProvider) buildRequestBody(req Request) map[string]any {
	messages := make([]map[string]string, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]string{
			"role":    string(msg.Role),
			"content": msg.Content,
		}
	}

	body := map[string]any{
		"model":    p.cfg.Model,
		"messages": messages,
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	body["max_tokens"] = maxTokens

	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	if req.JSONSchema != nil {
		body["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "extraction_result",
				"schema": req.JSONSchema,
				"strict": req.StrictMode,
			},
		}
	}

	return body
}

func (p *HeliconeProvider) parseResponse(body []byte, duration time.Duration) (*Response, error) {
	// OpenAI-compatible response format
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
		ID    string `json:"id"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &Response{
		Content:      result.Choices[0].Message.Content,
		FinishReason: result.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
		},
		Model:        result.Model,
		GenerationID: result.ID,
		Duration:     duration,
	}, nil
}

// ListModels fetches all available models from Helicone's public model registry.
// Results are cached for the configured TTL.
func (p *HeliconeProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
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
func (p *HeliconeProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
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

// SupportsGenerationCost returns true - Helicone supports generation cost lookup.
func (p *HeliconeProvider) SupportsGenerationCost() bool {
	return true
}

// GetGenerationCost fetches the actual cost from Helicone for a generation.
// Uses the Helicone Request API: GET /v1/request/{requestId}
func (p *HeliconeProvider) GetGenerationCost(ctx context.Context, generationID string) (float64, error) {
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

func (p *HeliconeProvider) fetchGenerationCost(ctx context.Context, generationID string) (float64, error) {
	url := fmt.Sprintf("%s/v1/request/%s", HeliconeAPIBaseURL, generationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
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

	// Helicone returns costUSD in the request response
	var result struct {
		Data struct {
			CostUSD float64 `json:"costUSD"`
			Cost    float64 `json:"cost"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Prefer costUSD, fall back to cost
	if result.Data.CostUSD > 0 {
		return result.Data.CostUSD, nil
	}
	return result.Data.Cost, nil
}

// EstimateCost calculates estimated cost based on cached pricing.
func (p *HeliconeProvider) EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (float64, error) {
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
func (p *HeliconeProvider) estimateCostFallback(model string, inputTokens, outputTokens int) float64 {
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
	}

	inputCost := float64(inputTokens) * inputPricePer1M / 1_000_000
	outputCost := float64(outputTokens) * outputPricePer1M / 1_000_000
	return inputCost + outputCost
}

// ensureModelCache refreshes the model cache if stale.
func (p *HeliconeProvider) ensureModelCache(ctx context.Context) error {
	p.modelCacheMu.RLock()
	isStale := time.Since(p.modelCacheTime) > p.cacheTTL
	p.modelCacheMu.RUnlock()

	if !isStale {
		return nil
	}

	return p.refreshModelCache(ctx)
}

func (p *HeliconeProvider) refreshModelCache(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, HeliconeModelsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Models endpoint is public, no auth required
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

	var result heliconeModelsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse and cache
	cache := make(map[string]*ModelInfo, len(result.Data.Models))
	for _, m := range result.Data.Models {
		// Use first endpoint's pricing (primary provider)
		var promptPrice, completionPrice float64
		var maxOutput int
		if len(m.Endpoints) > 0 {
			pricing := m.Endpoints[0].Pricing
			promptPrice = pricing.Prompt
			completionPrice = pricing.Completion
		}
		if m.MaxOutput > 0 {
			maxOutput = m.MaxOutput
		}

		info := &ModelInfo{
			ID:                  m.ID,
			Name:                m.Name,
			Description:         fmt.Sprintf("%s via Helicone", m.Author),
			ContextLength:       m.ContextLength,
			MaxCompletionTokens: maxOutput,
			PromptPrice:         promptPrice,
			CompletionPrice:     completionPrice,
			IsFree:              promptPrice == 0 && completionPrice == 0,
			Capabilities:        parseHeliconeCapabilities(m.Capabilities),
		}

		cache[m.ID] = info
	}

	p.modelCacheMu.Lock()
	p.modelCache = cache
	p.modelCacheTime = time.Now()
	p.modelCacheMu.Unlock()

	return nil
}

// Helicone API response types

type heliconeModelsResponse struct {
	Data struct {
		Models []heliconeModel `json:"models"`
		Total  int             `json:"total"`
	} `json:"data"`
}

type heliconeModel struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Author        string             `json:"author"`
	ContextLength int                `json:"contextLength"`
	MaxOutput     int                `json:"maxOutput"`
	Endpoints     []heliconeEndpoint `json:"endpoints"`
	Capabilities  []string           `json:"capabilities"`
}

type heliconeEndpoint struct {
	Provider     string           `json:"provider"`
	ProviderSlug string           `json:"providerSlug"`
	Pricing      heliconePricing  `json:"pricing"`
}

type heliconePricing struct {
	Prompt     float64 `json:"prompt"`
	Completion float64 `json:"completion"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Thinking   float64 `json:"thinking"`
}

func parseHeliconeCapabilities(caps []string) ModelCapabilities {
	result := ModelCapabilities{
		SupportsStreaming: true, // Helicone always supports streaming
	}

	for _, c := range caps {
		switch strings.ToLower(c) {
		case "structured_outputs", "json_mode":
			result.SupportsStructuredOutputs = true
		case "tools", "function_calling":
			result.SupportsTools = true
		case "reasoning", "thinking":
			result.SupportsReasoning = true
		case "vision", "image":
			result.SupportsVision = true
		case "response_format":
			result.SupportsResponseFormat = true
		}
	}

	return result
}

// Ensure HeliconeProvider implements all interfaces
var (
	_ Provider          = (*HeliconeProvider)(nil)
	_ ModelLister       = (*HeliconeProvider)(nil)
	_ ModelInfoProvider = (*HeliconeProvider)(nil)
	_ CostTracker       = (*HeliconeProvider)(nil)
	_ CostEstimator     = (*HeliconeProvider)(nil)
)

// ListHeliconeModels fetches all available models from Helicone's public model registry.
// This is a standalone function that doesn't require a provider instance since the
// models endpoint is public (no auth required).
func ListHeliconeModels(ctx context.Context) ([]ModelInfo, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, HeliconeModelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Models endpoint is public, no auth required
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result heliconeModelsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse models
	models := make([]ModelInfo, 0, len(result.Data.Models))
	for _, m := range result.Data.Models {
		// Use first endpoint's pricing (primary provider)
		var promptPrice, completionPrice float64
		var maxOutput int
		if len(m.Endpoints) > 0 {
			pricing := m.Endpoints[0].Pricing
			promptPrice = pricing.Prompt
			completionPrice = pricing.Completion
		}
		if m.MaxOutput > 0 {
			maxOutput = m.MaxOutput
		}

		info := ModelInfo{
			ID:                  m.ID,
			Name:                m.Name,
			Description:         fmt.Sprintf("%s via Helicone", m.Author),
			ContextLength:       m.ContextLength,
			MaxCompletionTokens: maxOutput,
			PromptPrice:         promptPrice,
			CompletionPrice:     completionPrice,
			IsFree:              promptPrice == 0 && completionPrice == 0,
			Capabilities:        parseHeliconeCapabilities(m.Capabilities),
		}

		models = append(models, info)
	}

	return models, nil
}
