package inference

import (
	"context"
	"fmt"

	"github.com/jmylchreest/refyne/pkg/llm"
)

// Remote wraps an existing pkg/llm Provider as an Inferencer.
// This bridges the existing provider implementations to the new interface.
type Remote struct {
	provider llm.Provider
	name     string
}

// RemoteOption configures a Remote inferencer.
type RemoteOption func(*remoteConfig)

type remoteConfig struct {
	apiKey         string
	model          string
	baseURL        string
	httpReferer    string
	appTitle       string
	targetProvider string
	targetAPIKey   string
}

// WithAPIKey sets the API key for the remote provider.
func WithAPIKey(key string) RemoteOption {
	return func(c *remoteConfig) { c.apiKey = key }
}

// WithModel sets the model name.
func WithModel(model string) RemoteOption {
	return func(c *remoteConfig) { c.model = model }
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) RemoteOption {
	return func(c *remoteConfig) { c.baseURL = url }
}

// WithHTTPReferer sets the HTTP referer for OpenRouter attribution.
func WithHTTPReferer(referer string) RemoteOption {
	return func(c *remoteConfig) { c.httpReferer = referer }
}

// WithAppTitle sets the app title for OpenRouter attribution.
func WithAppTitle(title string) RemoteOption {
	return func(c *remoteConfig) { c.appTitle = title }
}

// WithTargetProvider sets the target provider for proxy backends (e.g., Helicone).
func WithTargetProvider(provider, apiKey string) RemoteOption {
	return func(c *remoteConfig) {
		c.targetProvider = provider
		c.targetAPIKey = apiKey
	}
}

// NewRemote creates a Remote inferencer wrapping an existing pkg/llm provider.
func NewRemote(providerName string, opts ...RemoteOption) (*Remote, error) {
	cfg := &remoteConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	pcfg := llm.DefaultProviderConfig()
	if cfg.apiKey != "" {
		pcfg.APIKey = cfg.apiKey
	}
	if cfg.model != "" {
		pcfg.Model = cfg.model
	}
	if cfg.baseURL != "" {
		pcfg.BaseURL = cfg.baseURL
	}
	if cfg.httpReferer != "" {
		pcfg.HTTPReferer = cfg.httpReferer
	}
	if cfg.appTitle != "" {
		pcfg.AppTitle = cfg.appTitle
	}
	if cfg.targetProvider != "" {
		pcfg.TargetProvider = cfg.targetProvider
		pcfg.TargetAPIKey = cfg.targetAPIKey
	}

	provider, err := llm.NewProvider(providerName, pcfg)
	if err != nil {
		return nil, fmt.Errorf("create %s provider: %w", providerName, err)
	}

	return &Remote{
		provider: provider,
		name:     providerName,
	}, nil
}

// NewRemoteFromProvider wraps an existing pkg/llm.Provider directly.
func NewRemoteFromProvider(p llm.Provider) *Remote {
	return &Remote{
		provider: p,
		name:     p.Name(),
	}
}

// Infer sends a request to the remote provider.
func (r *Remote) Infer(ctx context.Context, req Request) (*Response, error) {
	// Convert inference.Request → llm.Request
	llmReq := llm.Request{
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		JSONSchema:  req.JSONSchema,
		StrictMode:  req.StrictMode,
	}

	for _, msg := range req.Messages {
		m := llm.Message{
			Role:    llm.Role(msg.Role),
			Content: msg.Content,
		}
		if msg.CacheControl != "" {
			m.CacheControl = llm.CacheControlType(msg.CacheControl)
		}
		llmReq.Messages = append(llmReq.Messages, m)
	}

	// Execute via the wrapped provider
	llmResp, err := r.provider.Execute(ctx, llmReq)
	if err != nil {
		return nil, err
	}

	// Convert llm.Response → inference.Response
	return &Response{
		Content:      llmResp.Content,
		Usage:        Usage{InputTokens: llmResp.Usage.InputTokens, OutputTokens: llmResp.Usage.OutputTokens},
		Model:        llmResp.Model,
		Provider:     r.name,
		Duration:     llmResp.Duration,
		FinishReason: llmResp.FinishReason,
		GenerationID: llmResp.GenerationID,
		Cost:         llmResp.Cost,
		CostIncluded: llmResp.CostIncluded,
	}, nil
}

// Name returns the provider name.
func (r *Remote) Name() string { return r.name }

// Available returns true (remote providers are stateless and always available).
func (r *Remote) Available() bool { return true }

// Close is a no-op for remote providers.
func (r *Remote) Close() error { return nil }

// Provider returns the underlying llm.Provider for advanced usage.
func (r *Remote) Provider() llm.Provider { return r.provider }
