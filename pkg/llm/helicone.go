package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// HeliconeCloudBaseURL is the default Helicone cloud gateway URL.
	HeliconeCloudBaseURL = "https://ai-gateway.helicone.ai"
)

// HeliconeProvider implements Provider for Helicone.
// Supports both cloud (managed keys) and self-hosted (proxy) modes.
type HeliconeProvider struct {
	cfg        ProviderConfig
	httpClient *http.Client
	selfHosted bool // True if using custom BaseURL (self-hosted mode)
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

// ListModels returns available models through Helicone.
// Helicone Cloud provides access to models from multiple providers.
func (p *HeliconeProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return []ModelInfo{
		{
			ID:            "gpt-4o",
			Name:          "GPT-4o",
			Description:   "Most capable GPT-4 model via Helicone",
			ContextLength: 128000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsResponseFormat:    true,
				SupportsVision:            true,
			},
		},
		{
			ID:            "gpt-4o-mini",
			Name:          "GPT-4o Mini",
			Description:   "Fast and affordable via Helicone",
			ContextLength: 128000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsResponseFormat:    true,
				SupportsVision:            true,
			},
		},
		{
			ID:            "gpt-4-turbo",
			Name:          "GPT-4 Turbo",
			Description:   "GPT-4 Turbo via Helicone",
			ContextLength: 128000,
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true,
				SupportsTools:             true,
				SupportsStreaming:         true,
				SupportsResponseFormat:    true,
				SupportsVision:            true,
			},
		},
		{
			ID:            "claude-3-5-sonnet-20241022",
			Name:          "Claude 3.5 Sonnet",
			Description:   "Anthropic's Claude 3.5 Sonnet via Helicone",
			ContextLength: 200000,
			Capabilities: ModelCapabilities{
				SupportsTools:     true,
				SupportsStreaming: true,
			},
		},
		{
			ID:            "claude-3-haiku-20240307",
			Name:          "Claude 3 Haiku",
			Description:   "Anthropic's Claude 3 Haiku via Helicone",
			ContextLength: 200000,
			Capabilities: ModelCapabilities{
				SupportsTools:     true,
				SupportsStreaming: true,
			},
		},
		{
			ID:            "gemini-1.5-pro",
			Name:          "Gemini 1.5 Pro",
			Description:   "Google's Gemini 1.5 Pro via Helicone",
			ContextLength: 1000000,
			Capabilities: ModelCapabilities{
				SupportsTools:     true,
				SupportsStreaming: true,
			},
		},
	}, nil
}

// GetModelInfo returns info for a specific model through Helicone.
func (p *HeliconeProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
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

// Ensure HeliconeProvider implements required interfaces
var (
	_ Provider          = (*HeliconeProvider)(nil)
	_ ModelLister       = (*HeliconeProvider)(nil)
	_ ModelInfoProvider = (*HeliconeProvider)(nil)
)
