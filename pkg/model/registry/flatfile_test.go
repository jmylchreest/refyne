package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testIndex is a helper that returns a valid JSON index with test models.
func testIndex() string {
	return `{
  "models": [
    {
      "name": "qwen3-0.6b-q4_k_m",
      "description": "Qwen3 0.6B Q4_K_M quantization",
      "version": "1.0",
      "format": "gguf",
      "quantization": "q4_k_m",
      "size_bytes": 500000000,
      "sha256": "",
      "download_url": "https://example.com/qwen3-q4.gguf",
      "tags": ["recipe", "extraction", "small"]
    },
    {
      "name": "qwen3-0.6b-q8_0",
      "description": "Qwen3 0.6B Q8_0 quantization",
      "version": "1.0",
      "format": "gguf",
      "quantization": "q8_0",
      "size_bytes": 900000000,
      "sha256": "",
      "download_url": "https://example.com/qwen3-q8.gguf",
      "tags": ["recipe", "extraction"]
    },
    {
      "name": "llama3-8b-safetensors",
      "description": "Llama 3 8B safetensors",
      "version": "1.0",
      "format": "safetensors",
      "quantization": "",
      "size_bytes": 16000000000,
      "sha256": "",
      "download_url": "https://example.com/llama3-8b.safetensors",
      "tags": ["general", "large"]
    }
  ]
}`
}

// writeTestIndex writes the test index JSON to a file in the given directory.
func writeTestIndex(t *testing.T, dir string) string {
	t.Helper()
	indexPath := filepath.Join(dir, "models.json")
	if err := os.WriteFile(indexPath, []byte(testIndex()), 0o644); err != nil {
		t.Fatalf("failed to write test index: %v", err)
	}
	return indexPath
}

// --- NewFlatFile ---

func TestNewFlatFile_ValidIndex(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)

	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}
	if ff == nil {
		t.Fatal("NewFlatFile() returned nil")
	}
	if len(ff.models) != 3 {
		t.Errorf("expected 3 models, got %d", len(ff.models))
	}
}

func TestNewFlatFile_InvalidPath(t *testing.T) {
	_, err := NewFlatFile("/nonexistent/path/index.json", t.TempDir())
	if err == nil {
		t.Fatal("NewFlatFile() expected error for invalid path, got nil")
	}
}

func TestNewFlatFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(invalidPath, []byte("this is not json{{{"), 0o644); err != nil {
		t.Fatalf("failed to write invalid JSON file: %v", err)
	}

	_, err := NewFlatFile(invalidPath, dir)
	if err == nil {
		t.Fatal("NewFlatFile() expected error for invalid JSON, got nil")
	}
}

func TestNewFlatFile_EmptyModelsList(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(indexPath, []byte(`{"models": []}`), 0o644); err != nil {
		t.Fatalf("failed to write empty index: %v", err)
	}

	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}
	if len(ff.models) != 0 {
		t.Errorf("expected 0 models, got %d", len(ff.models))
	}
}

func TestNewFlatFile_DefaultCacheDir(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)

	ff, err := NewFlatFile(indexPath, "")
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}
	if ff.cacheDir == "" {
		t.Error("expected cacheDir to be set to default when empty string passed")
	}
}

// --- List ---

func TestFlatFile_List_NoFilters(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	models, err := ff.List(context.Background())
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(models) != 3 {
		t.Errorf("List() with no filters: expected 3 models, got %d", len(models))
	}
}

func TestFlatFile_List_WithFormatFilter(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	tests := []struct {
		format   string
		expected int
	}{
		{format: "gguf", expected: 2},
		{format: "safetensors", expected: 1},
		{format: "onnx", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			models, err := ff.List(context.Background(), WithFormat(tt.format))
			if err != nil {
				t.Fatalf("List() returned unexpected error: %v", err)
			}
			if len(models) != tt.expected {
				t.Errorf("List(WithFormat(%q)): expected %d models, got %d", tt.format, tt.expected, len(models))
			}
		})
	}
}

func TestFlatFile_List_WithQuantizationFilter(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	tests := []struct {
		quant    string
		expected int
	}{
		{quant: "q4_k_m", expected: 1},
		{quant: "q8_0", expected: 1},
		{quant: "q2_k", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.quant, func(t *testing.T) {
			models, err := ff.List(context.Background(), WithQuantization(tt.quant))
			if err != nil {
				t.Fatalf("List() returned unexpected error: %v", err)
			}
			if len(models) != tt.expected {
				t.Errorf("List(WithQuantization(%q)): expected %d models, got %d", tt.quant, tt.expected, len(models))
			}
		})
	}
}

func TestFlatFile_List_WithTagFilter(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	tests := []struct {
		name     string
		tags     []string
		expected int
	}{
		{name: "recipe tag", tags: []string{"recipe"}, expected: 2},
		{name: "general tag", tags: []string{"general"}, expected: 1},
		{name: "extraction+small", tags: []string{"extraction", "small"}, expected: 1},
		{name: "nonexistent tag", tags: []string{"nonexistent"}, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			models, err := ff.List(context.Background(), WithTags(tt.tags...))
			if err != nil {
				t.Fatalf("List() returned unexpected error: %v", err)
			}
			if len(models) != tt.expected {
				t.Errorf("List(WithTags(%v)): expected %d models, got %d", tt.tags, tt.expected, len(models))
			}
		})
	}
}

func TestFlatFile_List_CombinedFilters(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	models, err := ff.List(context.Background(), WithFormat("gguf"), WithQuantization("q4_k_m"))
	if err != nil {
		t.Fatalf("List() returned unexpected error: %v", err)
	}
	if len(models) != 1 {
		t.Errorf("List(gguf+q4_k_m): expected 1 model, got %d", len(models))
	}
	if len(models) > 0 && models[0].Name != "qwen3-0.6b-q4_k_m" {
		t.Errorf("expected model name %q, got %q", "qwen3-0.6b-q4_k_m", models[0].Name)
	}
}

// --- Get ---

func TestFlatFile_Get_ExistingModel(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	model, err := ff.Get(context.Background(), "qwen3-0.6b-q4_k_m")
	if err != nil {
		t.Fatalf("Get() returned unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("Get() returned nil model")
	}
	if model.Name != "qwen3-0.6b-q4_k_m" {
		t.Errorf("Get() name: expected %q, got %q", "qwen3-0.6b-q4_k_m", model.Name)
	}
	if model.Format != "gguf" {
		t.Errorf("Get() format: expected %q, got %q", "gguf", model.Format)
	}
	if model.Quantization != "q4_k_m" {
		t.Errorf("Get() quantization: expected %q, got %q", "q4_k_m", model.Quantization)
	}
}

func TestFlatFile_Get_MissingModel(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	model, err := ff.Get(context.Background(), "nonexistent-model")
	if model != nil {
		t.Errorf("Get() expected nil model for missing name, got %v", model)
	}
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("Get() expected ErrModelNotFound, got: %v", err)
	}
}

func TestFlatFile_Get_AllModels(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	names := []string{"qwen3-0.6b-q4_k_m", "qwen3-0.6b-q8_0", "llama3-8b-safetensors"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			model, err := ff.Get(context.Background(), name)
			if err != nil {
				t.Fatalf("Get(%q) returned unexpected error: %v", name, err)
			}
			if model.Name != name {
				t.Errorf("Get(%q) returned model with name %q", name, model.Name)
			}
		})
	}
}

// --- Resolve ---

func TestFlatFile_Resolve_PreferredQuantization(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	model, err := ff.Resolve(context.Background(), WithPreferredQuantization("q8_0"))
	if err != nil {
		t.Fatalf("Resolve() returned unexpected error: %v", err)
	}
	if model.Quantization != "q8_0" {
		t.Errorf("Resolve(q8_0): expected quantization %q, got %q", "q8_0", model.Quantization)
	}
	if model.Name != "qwen3-0.6b-q8_0" {
		t.Errorf("Resolve(q8_0): expected name %q, got %q", "qwen3-0.6b-q8_0", model.Name)
	}
}

func TestFlatFile_Resolve_MaxSizeConstraint(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	// Max size 600MB -- should exclude models > 600MB.
	model, err := ff.Resolve(context.Background(), WithMaxSize(600_000_000))
	if err != nil {
		t.Fatalf("Resolve() returned unexpected error: %v", err)
	}
	if model.SizeBytes > 600_000_000 {
		t.Errorf("Resolve(MaxSize=600MB): returned model with size %d, expected <= 600000000", model.SizeBytes)
	}
}

func TestFlatFile_Resolve_NoMatch(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	// Max size 1 byte -- nothing should match.
	_, err = ff.Resolve(context.Background(), WithMaxSize(1))
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("Resolve(MaxSize=1): expected ErrModelNotFound, got: %v", err)
	}
}

func TestFlatFile_Resolve_NoOptions_ReturnsFirst(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	model, err := ff.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() returned unexpected error: %v", err)
	}
	// With no constraints, should return the first model in the list.
	if model.Name != "qwen3-0.6b-q4_k_m" {
		t.Errorf("Resolve() with no options: expected first model %q, got %q", "qwen3-0.6b-q4_k_m", model.Name)
	}
}

func TestFlatFile_Resolve_PreferredQuantization_NotAvailable_FallsBack(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	// Preferred q2_k does not exist; should fall back to the first model.
	model, err := ff.Resolve(context.Background(), WithPreferredQuantization("q2_k"))
	if err != nil {
		t.Fatalf("Resolve() returned unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("Resolve() returned nil model")
	}
	// Should return the first available model as fallback.
	if model.Name != "qwen3-0.6b-q4_k_m" {
		t.Errorf("Resolve(q2_k fallback): expected %q, got %q", "qwen3-0.6b-q4_k_m", model.Name)
	}
}

func TestFlatFile_Resolve_EmptyRegistry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(indexPath, []byte(`{"models": []}`), 0o644); err != nil {
		t.Fatalf("failed to write empty index: %v", err)
	}

	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	_, err = ff.Resolve(context.Background())
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("Resolve() on empty registry: expected ErrModelNotFound, got: %v", err)
	}
}

// --- hasAllTags ---

func TestHasAllTags(t *testing.T) {
	tests := []struct {
		name      string
		modelTags []string
		required  []string
		expected  bool
	}{
		{
			name:      "all tags present",
			modelTags: []string{"recipe", "extraction", "small"},
			required:  []string{"recipe", "extraction"},
			expected:  true,
		},
		{
			name:      "missing tag",
			modelTags: []string{"recipe", "extraction"},
			required:  []string{"recipe", "large"},
			expected:  false,
		},
		{
			name:      "no required tags",
			modelTags: []string{"recipe", "extraction"},
			required:  []string{},
			expected:  true,
		},
		{
			name:      "no model tags with requirements",
			modelTags: []string{},
			required:  []string{"recipe"},
			expected:  false,
		},
		{
			name:      "both empty",
			modelTags: []string{},
			required:  []string{},
			expected:  true,
		},
		{
			name:      "case insensitive match",
			modelTags: []string{"Recipe", "EXTRACTION"},
			required:  []string{"recipe", "extraction"},
			expected:  true,
		},
		{
			name:      "case insensitive required",
			modelTags: []string{"recipe", "extraction"},
			required:  []string{"Recipe", "EXTRACTION"},
			expected:  true,
		},
		{
			name:      "mixed case match",
			modelTags: []string{"Recipe", "extraction"},
			required:  []string{"RECIPE", "Extraction"},
			expected:  true,
		},
		{
			name:      "exact single tag",
			modelTags: []string{"recipe"},
			required:  []string{"recipe"},
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasAllTags(tt.modelTags, tt.required)
			if got != tt.expected {
				t.Errorf("hasAllTags(%v, %v) = %v, want %v", tt.modelTags, tt.required, got, tt.expected)
			}
		})
	}
}

// --- verifySHA256 ---

func TestVerifySHA256_CorrectHash(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.bin")
	content := []byte("hello world sha256 test content")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Compute expected hash.
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])

	ok, err := verifySHA256(filePath, expectedHash)
	if err != nil {
		t.Fatalf("verifySHA256() returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("verifySHA256() should return true for correct hash")
	}
}

func TestVerifySHA256_IncorrectHash(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.bin")
	content := []byte("hello world sha256 test content")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	ok, err := verifySHA256(filePath, wrongHash)
	if err != nil {
		t.Fatalf("verifySHA256() returned unexpected error: %v", err)
	}
	if ok {
		t.Error("verifySHA256() should return false for incorrect hash")
	}
}

func TestVerifySHA256_NonexistentFile(t *testing.T) {
	_, err := verifySHA256("/nonexistent/file.bin", "abc123")
	if err == nil {
		t.Error("verifySHA256() should return error for nonexistent file")
	}
}

func TestVerifySHA256_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(filePath, []byte{}, 0o644); err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	// SHA256 of empty content.
	h := sha256.Sum256([]byte{})
	expectedHash := hex.EncodeToString(h[:])

	ok, err := verifySHA256(filePath, expectedHash)
	if err != nil {
		t.Fatalf("verifySHA256() returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("verifySHA256() should return true for correct hash of empty file")
	}
}

func TestVerifySHA256_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.bin")
	content := []byte("case insensitive test")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	h := sha256.Sum256(content)
	lowerHash := hex.EncodeToString(h[:])
	upperHash := strings.ToUpper(lowerHash)

	ok, err := verifySHA256(filePath, upperHash)
	if err != nil {
		t.Fatalf("verifySHA256() returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("verifySHA256() should be case-insensitive")
	}
}

// Need to import strings for the case test above.
func TestVerifySHA256_UpperCaseExpected(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "testfile.bin")
	content := []byte("upper case hash test")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	h := sha256.Sum256(content)
	// Use uppercase hex encoding.
	hash := hex.EncodeToString(h[:])
	upperHash := ""
	for _, c := range hash {
		if c >= 'a' && c <= 'f' {
			upperHash += string(c - 32)
		} else {
			upperHash += string(c)
		}
	}

	ok, err := verifySHA256(filePath, upperHash)
	if err != nil {
		t.Fatalf("verifySHA256() returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("verifySHA256() should accept uppercase hex hash")
	}
}

// --- Registry interface compliance ---

func TestFlatFile_ImplementsRegistry(t *testing.T) {
	dir := t.TempDir()
	indexPath := writeTestIndex(t, dir)
	ff, err := NewFlatFile(indexPath, dir)
	if err != nil {
		t.Fatalf("NewFlatFile() returned unexpected error: %v", err)
	}

	var _ Registry = ff
}
