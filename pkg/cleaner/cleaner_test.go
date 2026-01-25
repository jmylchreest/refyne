package cleaner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readTestdata reads a file from the testdata directory
func readTestdata(t *testing.T, filename string) string {
	t.Helper()
	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read testdata %s: %v", filename, err)
	}
	return string(data)
}

// --- NoopCleaner Tests ---

func TestNoopCleaner_Clean(t *testing.T) {
	c := NewNoop()

	tests := []struct {
		name  string
		input string
	}{
		{"empty_string", ""},
		{"plain_text", "Hello, World!"},
		{"html_content", "<html><body><h1>Title</h1></body></html>"},
		{"whitespace", "  \n\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Clean(tt.input)
			if err != nil {
				t.Errorf("Clean() error = %v, want nil", err)
			}
			if got != tt.input {
				t.Errorf("Clean() = %q, want %q", got, tt.input)
			}
		})
	}
}

func TestNoopCleaner_Name(t *testing.T) {
	c := NewNoop()
	if got := c.Name(); got != "noop" {
		t.Errorf("Name() = %q, want %q", got, "noop")
	}
}

// Markdown tests are in markdown_test.go (requires -tags markdown)
// Trafilatura tests are in trafilatura_test.go (requires -tags trafilatura)

// --- ChainCleaner Tests ---

func TestChainCleaner_Empty(t *testing.T) {
	c := NewChain()

	input := "unchanged content"
	got, err := c.Clean(input)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if got != input {
		t.Errorf("Clean() = %q, want %q", got, input)
	}
}

func TestChainCleaner_SingleCleaner(t *testing.T) {
	c := NewChain(NewNoop())

	input := "test content"
	got, err := c.Clean(input)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if got != input {
		t.Errorf("Clean() = %q, want %q", got, input)
	}
}

func TestChainCleaner_MultipleCleanes_Order(t *testing.T) {
	// Create a chain with markdown cleaner
	md := NewMarkdown()
	c := NewChain(md)

	html := `<h1>Title</h1><p>Content</p>`
	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// If markdown is not available (stub), it passes through unchanged
	if !md.IsAvailable() {
		if got != html {
			t.Errorf("expected passthrough with stub, got %q", got)
		}
		return
	}

	// If markdown is available, expect actual conversion
	if !strings.Contains(got, "# Title") {
		t.Errorf("expected markdown output, got %q", got)
	}
}

// errorCleaner is a test cleaner that always returns an error
type errorCleaner struct{}

func (c *errorCleaner) Clean(html string) (string, error) {
	return "", errors.New("test error")
}

func (c *errorCleaner) Name() string {
	return "error"
}

func TestChainCleaner_ErrorPropagation(t *testing.T) {
	c := NewChain(NewNoop(), &errorCleaner{}, NewMarkdown())

	_, err := c.Clean("test")
	if err == nil {
		t.Fatal("expected error to propagate")
	}

	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("expected error containing 'test error', got %v", err)
	}
}

func TestChainCleaner_Name(t *testing.T) {
	tests := []struct {
		name     string
		cleaners []Cleaner
		want     string
	}{
		{"empty", []Cleaner{}, "chain()"},
		{"single", []Cleaner{NewNoop()}, "chain(noop)"},
		{"double", []Cleaner{NewNoop(), NewMarkdown()}, "chain(noop->markdown)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChain(tt.cleaners...)
			if got := c.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Integration tests with trafilatura are in trafilatura_test.go (requires -tags trafilatura)

// --- Option Tests ---

func TestWithStripLinks(t *testing.T) {
	cfg := &markdownConfig{}
	WithStripLinks(true)(cfg)

	if !cfg.StripLinks {
		t.Error("WithStripLinks(true) did not set StripLinks")
	}

	WithStripLinks(false)(cfg)
	if cfg.StripLinks {
		t.Error("WithStripLinks(false) did not unset StripLinks")
	}
}

func TestWithStripImages(t *testing.T) {
	cfg := &markdownConfig{}
	WithStripImages(true)(cfg)

	if !cfg.StripImages {
		t.Error("WithStripImages(true) did not set StripImages")
	}
}
