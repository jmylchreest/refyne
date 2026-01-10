package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider communicates with a local Ollama instance.
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

	client := &http.Client{}
	if cfg.Timeout > 0 {
		client.Timeout = cfg.Timeout
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  client,
	}, nil
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Format   json.RawMessage `json:"format,omitempty"`
	Stream   bool            `json:"stream"`
	Options  ollamaOptions   `json:"options,omitempty"`
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
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	PromptEvalCount int           `json:"prompt_eval_count"`
	EvalCount       int           `json:"eval_count"`
}

// Complete sends a completion request to Ollama.
func (p *OllamaProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
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
			return CompletionResponse{}, fmt.Errorf("failed to marshal JSON schema: %w", err)
		}
		ollamaReq.Format = schemaBytes
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return CompletionResponse{}, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return CompletionResponse{
		Content:      ollamaResp.Message.Content,
		FinishReason: "stop",
		Usage: Usage{
			InputTokens:  ollamaResp.PromptEvalCount,
			OutputTokens: ollamaResp.EvalCount,
		},
	}, nil
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// SupportsJSONSchema returns true as Ollama 0.5+ supports structured outputs.
func (p *OllamaProvider) SupportsJSONSchema() bool {
	return true
}

func init() {
	RegisterProvider("ollama", func(cfg ProviderConfig) (Provider, error) {
		return NewOllamaProvider(cfg)
	})
}
