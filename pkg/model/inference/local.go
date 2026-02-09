package inference

import (
	"context"
	"errors"
	"sync"
)

// Local provides inference using a local GGUF model via llama.cpp.
// This is a stub that will be implemented once cgo bindings are added.
type Local struct {
	modelPath string
	mu        sync.Mutex
	loaded    bool
	opts      localConfig
}

// LocalOption configures a Local inferencer.
type LocalOption func(*localConfig)

type localConfig struct {
	nGPULayers int
	nCtx       int
	nThreads   int
}

// WithGPULayers sets the number of layers to offload to GPU.
func WithGPULayers(n int) LocalOption {
	return func(c *localConfig) { c.nGPULayers = n }
}

// WithContextSize sets the context window size.
func WithContextSize(n int) LocalOption {
	return func(c *localConfig) { c.nCtx = n }
}

// WithThreads sets the number of CPU threads.
func WithThreads(n int) LocalOption {
	return func(c *localConfig) { c.nThreads = n }
}

// NewLocal creates a local GGUF inferencer.
// The model is not loaded until the first Infer call.
func NewLocal(modelPath string, opts ...LocalOption) (*Local, error) {
	cfg := localConfig{
		nGPULayers: 0,
		nCtx:       4096,
		nThreads:   4,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Local{
		modelPath: modelPath,
		opts:      cfg,
	}, nil
}

// Infer runs inference on the local model.
// TODO: Implement via cgo bindings to llama.cpp.
func (l *Local) Infer(_ context.Context, _ Request) (*Response, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return nil, errors.New("local GGUF inference not yet implemented: requires cgo bindings to llama.cpp")
}

// Name returns "local-gguf".
func (l *Local) Name() string { return "local-gguf" }

// Available returns true if the model file exists.
func (l *Local) Available() bool {
	// TODO: Check if model file exists and is loadable.
	return false
}

// Close unloads the model and releases resources.
func (l *Local) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loaded = false
	return nil
}

// ModelPath returns the path to the GGUF model file.
func (l *Local) ModelPath() string { return l.modelPath }
