package inference

import (
	"context"
	"strings"
	"testing"
)

// --- NewLocal defaults ---

func TestNewLocal_DefaultOptions(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	if l.opts.nGPULayers != 0 {
		t.Errorf("default nGPULayers: expected 0, got %d", l.opts.nGPULayers)
	}
	if l.opts.nCtx != 4096 {
		t.Errorf("default nCtx: expected 4096, got %d", l.opts.nCtx)
	}
	if l.opts.nThreads != 4 {
		t.Errorf("default nThreads: expected 4, got %d", l.opts.nThreads)
	}
	if l.modelPath != "/tmp/model.gguf" {
		t.Errorf("modelPath: expected %q, got %q", "/tmp/model.gguf", l.modelPath)
	}
	if l.loaded {
		t.Error("loaded: expected false for newly created Local")
	}
}

// --- NewLocal with custom options ---

func TestNewLocal_WithGPULayers(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf", WithGPULayers(32))
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}
	if l.opts.nGPULayers != 32 {
		t.Errorf("nGPULayers: expected 32, got %d", l.opts.nGPULayers)
	}
}

func TestNewLocal_WithContextSize(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf", WithContextSize(8192))
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}
	if l.opts.nCtx != 8192 {
		t.Errorf("nCtx: expected 8192, got %d", l.opts.nCtx)
	}
}

func TestNewLocal_WithThreads(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf", WithThreads(8))
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}
	if l.opts.nThreads != 8 {
		t.Errorf("nThreads: expected 8, got %d", l.opts.nThreads)
	}
}

func TestNewLocal_WithAllOptions(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf",
		WithGPULayers(16),
		WithContextSize(2048),
		WithThreads(12),
	)
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}
	if l.opts.nGPULayers != 16 {
		t.Errorf("nGPULayers: expected 16, got %d", l.opts.nGPULayers)
	}
	if l.opts.nCtx != 2048 {
		t.Errorf("nCtx: expected 2048, got %d", l.opts.nCtx)
	}
	if l.opts.nThreads != 12 {
		t.Errorf("nThreads: expected 12, got %d", l.opts.nThreads)
	}
}

func TestNewLocal_OptionsOverrideDefaults(t *testing.T) {
	// WithContextSize should override only nCtx, leaving other defaults intact.
	l, err := NewLocal("/tmp/model.gguf", WithContextSize(512))
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}
	if l.opts.nCtx != 512 {
		t.Errorf("nCtx: expected 512, got %d", l.opts.nCtx)
	}
	// nGPULayers and nThreads should retain defaults
	if l.opts.nGPULayers != 0 {
		t.Errorf("nGPULayers: expected default 0, got %d", l.opts.nGPULayers)
	}
	if l.opts.nThreads != 4 {
		t.Errorf("nThreads: expected default 4, got %d", l.opts.nThreads)
	}
}

// --- Infer ---

func TestLocal_Infer_ReturnsNotImplementedError(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	resp, err := l.Infer(context.Background(), Request{})
	if err == nil {
		t.Fatal("Infer() expected error, got nil")
	}
	if resp != nil {
		t.Errorf("Infer() expected nil response, got %v", resp)
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Infer() error should contain 'not yet implemented', got: %v", err)
	}
}

func TestLocal_Infer_ErrorContainsCgoReference(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	_, err = l.Infer(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	})
	if err == nil {
		t.Fatal("Infer() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "llama.cpp") {
		t.Errorf("Infer() error should reference llama.cpp, got: %v", err)
	}
}

// --- Name ---

func TestLocal_Name(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	name := l.Name()
	if name != "local-gguf" {
		t.Errorf("Name(): expected %q, got %q", "local-gguf", name)
	}
}

// --- Available ---

func TestLocal_Available_ReturnsFalse(t *testing.T) {
	l, err := NewLocal("/tmp/nonexistent-model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	if l.Available() {
		t.Error("Available(): expected false for stub implementation")
	}
}

// --- Close ---

func TestLocal_Close_Succeeds(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Errorf("Close() returned unexpected error: %v", err)
	}
}

func TestLocal_Close_SetsLoadedFalse(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	// Manually set loaded to true to simulate loaded state.
	l.mu.Lock()
	l.loaded = true
	l.mu.Unlock()

	if err := l.Close(); err != nil {
		t.Fatalf("Close() returned unexpected error: %v", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.loaded {
		t.Error("Close() should set loaded to false")
	}
}

func TestLocal_Close_Idempotent(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := l.Close(); err != nil {
			t.Errorf("Close() call %d returned unexpected error: %v", i+1, err)
		}
	}
}

// --- ModelPath ---

func TestLocal_ModelPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "absolute path", path: "/tmp/model.gguf"},
		{name: "relative path", path: "models/my-model.gguf"},
		{name: "nested path", path: "/home/user/.cache/refyne/models/qwen3-0.6b-q4_k_m.gguf"},
		{name: "empty path", path: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := NewLocal(tt.path)
			if err != nil {
				t.Fatalf("NewLocal() returned unexpected error: %v", err)
			}
			if l.ModelPath() != tt.path {
				t.Errorf("ModelPath(): expected %q, got %q", tt.path, l.ModelPath())
			}
		})
	}
}

// --- Inferencer interface compliance ---

func TestLocal_ImplementsInferencer(t *testing.T) {
	l, err := NewLocal("/tmp/model.gguf")
	if err != nil {
		t.Fatalf("NewLocal() returned unexpected error: %v", err)
	}

	// Compile-time check: *Local should satisfy the Inferencer interface.
	var _ Inferencer = l
}
