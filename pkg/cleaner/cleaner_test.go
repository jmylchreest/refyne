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

// --- MarkdownCleaner Tests ---

func TestMarkdownCleaner_Clean_BasicHTML(t *testing.T) {
	c := NewMarkdown()

	html := `<h1>Title</h1><p>A paragraph.</p>`

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if !strings.Contains(got, "# Title") {
		t.Errorf("expected markdown heading, got %q", got)
	}

	if !strings.Contains(got, "A paragraph.") {
		t.Errorf("expected paragraph text, got %q", got)
	}
}

func TestMarkdownCleaner_Clean_WithHeaders(t *testing.T) {
	c := NewMarkdown()

	html := `<h1>H1</h1><h2>H2</h2><h3>H3</h3>`

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if !strings.Contains(got, "# H1") {
		t.Errorf("expected # H1, got %q", got)
	}

	if !strings.Contains(got, "## H2") {
		t.Errorf("expected ## H2, got %q", got)
	}

	if !strings.Contains(got, "### H3") {
		t.Errorf("expected ### H3, got %q", got)
	}
}

func TestMarkdownCleaner_Clean_WithLists(t *testing.T) {
	c := NewMarkdown()

	html := `<ul><li>Item 1</li><li>Item 2</li></ul>`

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if !strings.Contains(got, "Item 1") && !strings.Contains(got, "Item 2") {
		t.Errorf("expected list items, got %q", got)
	}
}

func TestMarkdownCleaner_Clean_WithLinks(t *testing.T) {
	c := NewMarkdown()

	html := `<a href="https://example.com">Example Link</a>`

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// Markdown links should include both text and URL
	if !strings.Contains(got, "Example Link") {
		t.Errorf("expected link text, got %q", got)
	}

	if !strings.Contains(got, "example.com") {
		t.Errorf("expected link URL, got %q", got)
	}
}

func TestMarkdownCleaner_Clean_FromTestdata(t *testing.T) {
	c := NewMarkdown()

	html := readTestdata(t, "simple.html")

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// Check key content is preserved
	checks := []string{
		"Main Heading",
		"paragraph",
		"bold",
		"italic",
		"Second Heading",
		"First item",
		"link to example",
	}

	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("expected %q in output, got %q", check, got)
		}
	}
}

func TestMarkdownCleaner_Name(t *testing.T) {
	c := NewMarkdown()
	if got := c.Name(); got != "markdown" {
		t.Errorf("Name() = %q, want %q", got, "markdown")
	}
}

// --- cleanWhitespace Tests ---

func TestCleanWhitespace_MultipleBlankLines(t *testing.T) {
	input := "Line 1\n\n\n\n\nLine 2"
	got := cleanWhitespace(input)

	// Should collapse multiple blank lines to max 2
	blankCount := 0
	for _, line := range strings.Split(got, "\n") {
		if strings.TrimSpace(line) == "" {
			blankCount++
		}
	}

	if blankCount > 2 {
		t.Errorf("expected max 2 blank lines, got %d in %q", blankCount, got)
	}
}

func TestCleanWhitespace_LeadingTrailingSpace(t *testing.T) {
	input := "\n\n  Content  \n\n"
	got := cleanWhitespace(input)

	if strings.HasPrefix(got, "\n") || strings.HasPrefix(got, " ") {
		t.Errorf("expected no leading whitespace, got %q", got)
	}

	if strings.HasSuffix(got, "\n") || strings.HasSuffix(got, " ") {
		t.Errorf("expected no trailing whitespace, got %q", got)
	}
}

func TestCleanWhitespace_NoChange(t *testing.T) {
	input := "Line 1\n\nLine 2"
	got := cleanWhitespace(input)

	if got != input {
		t.Errorf("cleanWhitespace() = %q, want %q", got, input)
	}
}

func TestCleanWhitespace_Empty(t *testing.T) {
	got := cleanWhitespace("")
	if got != "" {
		t.Errorf("cleanWhitespace(\"\") = %q, want \"\"", got)
	}
}

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
	c := NewChain(NewMarkdown())

	html := `<h1>Title</h1><p>Content</p>`
	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

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
