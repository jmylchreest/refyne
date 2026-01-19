package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider communicates with a local Ollama instance.
// Ollama is free/self-hosted, so no cost tracking is needed.
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(cfg ProviderConfig) (*OllamaProvider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Format   json.RawMessage `json:"format,omitempty"`
	Stream   bool            `json:"stream"`
	Options  ollamaOptions   `json:"options"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaResponse struct {
	Model           string        `json:"model"`
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	PromptEvalCount int           `json:"prompt_eval_count"`
	EvalCount       int           `json:"eval_count"`
}

// Execute sends a completion request to Ollama.
func (p *OllamaProvider) Execute(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()

	messages := make([]ollamaMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, ollamaMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	ollamaReq := ollamaRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
		Options: ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	// Ollama supports JSON format constraint via the format field
	if req.JSONSchema != nil {
		schemaBytes, err := json.Marshal(req.JSONSchema)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON schema: %w", err)
		}
		ollamaReq.Format = schemaBytes
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Response{
		Content:      ollamaResp.Message.Content,
		FinishReason: "stop",
		Usage: Usage{
			InputTokens:  ollamaResp.PromptEvalCount,
			OutputTokens: ollamaResp.EvalCount,
		},
		Model:        ollamaResp.Model,
		Cost:         0, // Ollama is free/self-hosted
		CostIncluded: true,
		Duration:     time.Since(start),
	}, nil
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Model returns the configured model name.
func (p *OllamaProvider) Model() string {
	return p.model
}

// ListModels fetches available models from the local Ollama instance.
func (p *OllamaProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			Model      string `json:"model"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
			Details    struct {
				ParameterSize     string `json:"parameter_size"`
				QuantizationLevel string `json:"quantization_level"`
			} `json:"details"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]ModelInfo, 0, len(result.Models))
	for _, m := range result.Models {
		models = append(models, ModelInfo{
			ID:          m.Name,
			Name:        m.Name,
			Description: fmt.Sprintf("%s (%s)", m.Details.ParameterSize, m.Details.QuantizationLevel),
			IsFree:      true, // Ollama is always free
			Capabilities: ModelCapabilities{
				SupportsStructuredOutputs: true, // Ollama 0.5+ supports JSON schema
				SupportsStreaming:         true,
			},
		})
	}

	return models, nil
}

// GetModelInfo returns info for a specific Ollama model.
func (p *OllamaProvider) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
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

// EstimateCost always returns 0 for Ollama (free/self-hosted).
func (p *OllamaProvider) EstimateCost(ctx context.Context, modelID string, inputTokens, outputTokens int) (float64, error) {
	return 0, nil
}

// Ensure OllamaProvider implements required interfaces
var (
	_ Provider          = (*OllamaProvider)(nil)
	_ ModelLister       = (*OllamaProvider)(nil)
	_ ModelInfoProvider = (*OllamaProvider)(nil)
	_ CostEstimator     = (*OllamaProvider)(nil)
	// Note: Ollama does NOT implement CostTracker - it's free/self-hosted
)
