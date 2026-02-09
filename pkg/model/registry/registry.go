// Package registry provides model discovery, metadata, download, and cache management.
package registry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// Registry provides model discovery and cache management.
type Registry interface {
	// List returns all models matching the given options.
	List(ctx context.Context, opts ...ListOption) ([]ModelInfo, error)

	// Get returns metadata for a specific model by name.
	Get(ctx context.Context, name string) (*ModelInfo, error)

	// Resolve finds the best model matching the given constraints.
	Resolve(ctx context.Context, opts ...ResolveOption) (*ModelInfo, error)

	// EnsureCached downloads the model if not already cached and returns the local path.
	EnsureCached(ctx context.Context, name string) (localPath string, err error)
}

// ModelInfo contains metadata about a model.
type ModelInfo struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Version      string            `json:"version,omitempty"`
	Format       string            `json:"format"` // "gguf", "safetensors", etc.
	Quantization string            `json:"quantization,omitempty"` // "q2_k", "q4_k_m", "q8_0", etc.
	SizeBytes    int64             `json:"size_bytes,omitempty"`
	SHA256       string            `json:"sha256,omitempty"`
	DownloadURL  string            `json:"download_url,omitempty"`
	LocalPath    string            `json:"local_path,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ListOption configures a List call.
type ListOption func(*listConfig)

type listConfig struct {
	format       string
	quantization string
	tags         []string
}

// WithFormat filters models by format (e.g., "gguf").
func WithFormat(format string) ListOption {
	return func(c *listConfig) { c.format = format }
}

// WithQuantization filters models by quantization level.
func WithQuantization(q string) ListOption {
	return func(c *listConfig) { c.quantization = q }
}

// WithTags filters models that have all specified tags.
func WithTags(tags ...string) ListOption {
	return func(c *listConfig) { c.tags = tags }
}

// ResolveOption configures a Resolve call.
type ResolveOption func(*resolveConfig)

type resolveConfig struct {
	maxSizeBytes int64
	preferredQ   string
}

// WithMaxSize constrains the resolved model to the given size.
func WithMaxSize(bytes int64) ResolveOption {
	return func(c *resolveConfig) { c.maxSizeBytes = bytes }
}

// WithPreferredQuantization sets the preferred quantization level.
func WithPreferredQuantization(q string) ResolveOption {
	return func(c *resolveConfig) { c.preferredQ = q }
}

// DefaultCacheDir returns the default model cache directory.
// Respects REFYNE_MODEL_CACHE env var, otherwise uses ~/.cache/refyne/models/.
func DefaultCacheDir() string {
	if dir := os.Getenv("REFYNE_MODEL_CACHE"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "refyne", "models")
	}
	return filepath.Join(home, ".cache", "refyne", "models")
}

// ErrModelNotFound is returned when a requested model is not in the registry.
var ErrModelNotFound = errors.New("model not found")
