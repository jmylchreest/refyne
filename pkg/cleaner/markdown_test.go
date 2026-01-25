//go:build markdown

package cleaner

import (
	"strings"
	"testing"
)

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
	if strings.Count(got, "\n\n\n") > 0 {
		t.Errorf("expected at most 2 consecutive newlines, got %q", got)
	}

	if !strings.Contains(got, "Line 1") || !strings.Contains(got, "Line 2") {
		t.Errorf("expected content preserved, got %q", got)
	}
}

func TestCleanWhitespace_LeadingTrailingSpace(t *testing.T) {
	input := "\n\n  Content  \n\n"
	got := cleanWhitespace(input)

	if strings.HasPrefix(got, "\n") || strings.HasSuffix(got, "\n") {
		t.Errorf("expected trimmed output, got %q", got)
	}
}

func TestCleanWhitespace_EmptyString(t *testing.T) {
	got := cleanWhitespace("")
	if got != "" {
		t.Errorf("cleanWhitespace(\"\") = %q, want \"\"", got)
	}
}
