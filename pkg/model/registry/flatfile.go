package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FlatFile is a Registry backed by a JSON index file.
type FlatFile struct {
	indexPath string
	cacheDir  string
	models    []ModelInfo
}

// flatFileIndex is the JSON structure of the index file.
type flatFileIndex struct {
	Models []ModelInfo `json:"models"`
}

// NewFlatFile creates a FlatFile registry from a JSON index file.
func NewFlatFile(indexPath string, cacheDir string) (*FlatFile, error) {
	if cacheDir == "" {
		cacheDir = DefaultCacheDir()
	}

	ff := &FlatFile{
		indexPath: indexPath,
		cacheDir:  cacheDir,
	}

	if err := ff.load(); err != nil {
		return nil, err
	}

	return ff, nil
}

func (f *FlatFile) load() error {
	data, err := os.ReadFile(f.indexPath) //#nosec G304
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	var idx flatFileIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("parse index: %w", err)
	}

	f.models = idx.Models
	return nil
}

// List returns models matching the given options.
func (f *FlatFile) List(_ context.Context, opts ...ListOption) ([]ModelInfo, error) {
	cfg := &listConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var result []ModelInfo
	for _, m := range f.models {
		if cfg.format != "" && m.Format != cfg.format {
			continue
		}
		if cfg.quantization != "" && m.Quantization != cfg.quantization {
			continue
		}
		if len(cfg.tags) > 0 && !hasAllTags(m.Tags, cfg.tags) {
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

// Get returns a specific model by name.
func (f *FlatFile) Get(_ context.Context, name string) (*ModelInfo, error) {
	for i := range f.models {
		if f.models[i].Name == name {
			return &f.models[i], nil
		}
	}
	return nil, ErrModelNotFound
}

// Resolve finds the best model matching constraints.
func (f *FlatFile) Resolve(_ context.Context, opts ...ResolveOption) (*ModelInfo, error) {
	cfg := &resolveConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var best *ModelInfo
	for i := range f.models {
		m := &f.models[i]
		if cfg.maxSizeBytes > 0 && m.SizeBytes > cfg.maxSizeBytes {
			continue
		}
		if cfg.preferredQ != "" && m.Quantization == cfg.preferredQ {
			return m, nil
		}
		if best == nil {
			best = m
		}
	}
	if best == nil {
		return nil, ErrModelNotFound
	}
	return best, nil
}

// EnsureCached downloads the model if not cached and returns the local path.
func (f *FlatFile) EnsureCached(ctx context.Context, name string) (string, error) {
	model, err := f.Get(ctx, name)
	if err != nil {
		return "", err
	}

	// Check if already cached
	localPath := filepath.Join(f.cacheDir, name)
	if model.Format != "" {
		localPath = filepath.Join(f.cacheDir, name+"."+model.Format)
	}

	if _, err := os.Stat(localPath); err == nil {
		// Verify SHA256 if available
		if model.SHA256 != "" {
			if ok, _ := verifySHA256(localPath, model.SHA256); ok {
				return localPath, nil
			}
			// Hash mismatch — re-download
		} else {
			return localPath, nil
		}
	}

	// Download
	if model.DownloadURL == "" {
		return "", fmt.Errorf("model %s has no download URL", name)
	}

	if err := os.MkdirAll(f.cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	if err := downloadFile(ctx, model.DownloadURL, localPath); err != nil {
		return "", fmt.Errorf("download model: %w", err)
	}

	// Verify SHA256
	if model.SHA256 != "" {
		if ok, err := verifySHA256(localPath, model.SHA256); err != nil {
			return "", fmt.Errorf("verify sha256: %w", err)
		} else if !ok {
			os.Remove(localPath)
			return "", fmt.Errorf("sha256 mismatch for %s", name)
		}
	}

	return localPath, nil
}

func hasAllTags(modelTags, required []string) bool {
	tagSet := make(map[string]bool, len(modelTags))
	for _, t := range modelTags {
		tagSet[strings.ToLower(t)] = true
	}
	for _, r := range required {
		if !tagSet[strings.ToLower(r)] {
			return false
		}
	}
	return true
}

func verifySHA256(path, expected string) (bool, error) {
	f, err := os.Open(path) //#nosec G304
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(actual, expected), nil
}

func downloadFile(ctx context.Context, url, dest string) (retErr error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest) //#nosec G304
	if err != nil {
		return err
	}
	defer func() {
		out.Close()
		if retErr != nil {
			os.Remove(dest) // Clean up partial download on error.
		}
	}()

	_, retErr = io.Copy(out, resp.Body)
	return retErr
}
