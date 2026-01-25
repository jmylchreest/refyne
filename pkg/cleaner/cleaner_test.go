package cleaner

import (
	"errors"
	"os"
	"path/filepath"
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

func TestChainCleaner_MultipleCleaners(t *testing.T) {
	// Chain multiple noop cleaners - content should pass through unchanged
	c := NewChain(NewNoop(), NewNoop())

	input := "<h1>Title</h1><p>Content</p>"
	got, err := c.Clean(input)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if got != input {
		t.Errorf("Clean() = %q, want %q", got, input)
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
	c := NewChain(NewNoop(), &errorCleaner{}, NewNoop())

	_, err := c.Clean("test")
	if err == nil {
		t.Fatal("expected error to propagate")
	}

	if err.Error() != "test error" {
		t.Errorf("expected error 'test error', got %v", err)
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
		{"double", []Cleaner{NewNoop(), NewNoop()}, "chain(noop->noop)"},
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
