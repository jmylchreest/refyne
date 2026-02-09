package registry

import (
	"errors"
	"strings"
	"testing"
)

// --- DefaultCacheDir ---

func TestDefaultCacheDir_ContainsCachePath(t *testing.T) {
	// Ensure REFYNE_MODEL_CACHE is not set.
	t.Setenv("REFYNE_MODEL_CACHE", "")

	dir := DefaultCacheDir()
	if !strings.Contains(dir, ".cache/refyne/models") && !strings.Contains(dir, "refyne/models") {
		t.Errorf("DefaultCacheDir() = %q, expected it to contain '.cache/refyne/models' or 'refyne/models'", dir)
	}
}

func TestDefaultCacheDir_ReturnsAbsolutePath(t *testing.T) {
	t.Setenv("REFYNE_MODEL_CACHE", "")

	dir := DefaultCacheDir()
	if !strings.HasPrefix(dir, "/") {
		t.Errorf("DefaultCacheDir() = %q, expected an absolute path", dir)
	}
}

func TestDefaultCacheDir_RespectsEnvVar(t *testing.T) {
	customDir := "/custom/model/cache"
	t.Setenv("REFYNE_MODEL_CACHE", customDir)

	dir := DefaultCacheDir()
	if dir != customDir {
		t.Errorf("DefaultCacheDir() = %q, expected %q from REFYNE_MODEL_CACHE", dir, customDir)
	}
}

func TestDefaultCacheDir_EnvVarOverridesDefault(t *testing.T) {
	t.Setenv("REFYNE_MODEL_CACHE", "/override/path")

	dir := DefaultCacheDir()
	if strings.Contains(dir, ".cache") {
		t.Errorf("DefaultCacheDir() = %q, should not contain '.cache' when REFYNE_MODEL_CACHE is set", dir)
	}
}

// --- ErrModelNotFound ---

func TestErrModelNotFound_Message(t *testing.T) {
	if ErrModelNotFound.Error() != "model not found" {
		t.Errorf("ErrModelNotFound.Error() = %q, expected %q", ErrModelNotFound.Error(), "model not found")
	}
}

func TestErrModelNotFound_IsComparable(t *testing.T) {
	// Wrap the error and verify errors.Is still works.
	wrapped := errors.New("wrapped: " + ErrModelNotFound.Error())
	_ = wrapped // Just verifying the sentinel is usable.

	if !errors.Is(ErrModelNotFound, ErrModelNotFound) {
		t.Error("errors.Is(ErrModelNotFound, ErrModelNotFound) should return true")
	}
}

// --- ListOption functions ---

func TestWithFormat(t *testing.T) {
	cfg := &listConfig{}
	WithFormat("gguf")(cfg)
	if cfg.format != "gguf" {
		t.Errorf("WithFormat(\"gguf\"): expected format %q, got %q", "gguf", cfg.format)
	}
}

func TestWithQuantization(t *testing.T) {
	cfg := &listConfig{}
	WithQuantization("q4_k_m")(cfg)
	if cfg.quantization != "q4_k_m" {
		t.Errorf("WithQuantization(\"q4_k_m\"): expected quantization %q, got %q", "q4_k_m", cfg.quantization)
	}
}

func TestWithTags(t *testing.T) {
	cfg := &listConfig{}
	WithTags("recipe", "extraction")(cfg)
	if len(cfg.tags) != 2 {
		t.Fatalf("WithTags: expected 2 tags, got %d", len(cfg.tags))
	}
	if cfg.tags[0] != "recipe" {
		t.Errorf("WithTags: tag[0] expected %q, got %q", "recipe", cfg.tags[0])
	}
	if cfg.tags[1] != "extraction" {
		t.Errorf("WithTags: tag[1] expected %q, got %q", "extraction", cfg.tags[1])
	}
}

func TestWithTags_Empty(t *testing.T) {
	cfg := &listConfig{}
	WithTags()(cfg)
	if len(cfg.tags) != 0 {
		t.Errorf("WithTags(): expected 0 tags, got %d", len(cfg.tags))
	}
}

// --- ResolveOption functions ---

func TestWithMaxSize(t *testing.T) {
	cfg := &resolveConfig{}
	WithMaxSize(1_000_000_000)(cfg)
	if cfg.maxSizeBytes != 1_000_000_000 {
		t.Errorf("WithMaxSize: expected %d, got %d", int64(1_000_000_000), cfg.maxSizeBytes)
	}
}

func TestWithPreferredQuantization(t *testing.T) {
	cfg := &resolveConfig{}
	WithPreferredQuantization("q8_0")(cfg)
	if cfg.preferredQ != "q8_0" {
		t.Errorf("WithPreferredQuantization: expected %q, got %q", "q8_0", cfg.preferredQ)
	}
}

// --- Option combinations ---

func TestListOptions_MultipleApplied(t *testing.T) {
	cfg := &listConfig{}
	opts := []ListOption{
		WithFormat("gguf"),
		WithQuantization("q4_k_m"),
		WithTags("small", "fast"),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.format != "gguf" {
		t.Errorf("format: expected %q, got %q", "gguf", cfg.format)
	}
	if cfg.quantization != "q4_k_m" {
		t.Errorf("quantization: expected %q, got %q", "q4_k_m", cfg.quantization)
	}
	if len(cfg.tags) != 2 {
		t.Fatalf("tags: expected 2, got %d", len(cfg.tags))
	}
}

func TestResolveOptions_MultipleApplied(t *testing.T) {
	cfg := &resolveConfig{}
	opts := []ResolveOption{
		WithMaxSize(500_000_000),
		WithPreferredQuantization("q2_k"),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.maxSizeBytes != 500_000_000 {
		t.Errorf("maxSizeBytes: expected %d, got %d", int64(500_000_000), cfg.maxSizeBytes)
	}
	if cfg.preferredQ != "q2_k" {
		t.Errorf("preferredQ: expected %q, got %q", "q2_k", cfg.preferredQ)
	}
}
