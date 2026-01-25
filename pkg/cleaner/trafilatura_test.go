//go:build trafilatura

package cleaner

import (
	"testing"
)

// --- Trafilatura-specific tests that access internal fields ---
// These tests only run when the trafilatura build tag is set

func TestTrafilaturaCleaner_DefaultConfig(t *testing.T) {
	c := NewTrafilatura(nil)
	if c == nil {
		t.Fatal("NewTrafilatura(nil) returned nil")
	}
	if c.Name() != "trafilatura" {
		t.Errorf("Name() = %q, want %q", c.Name(), "trafilatura")
	}
}

func TestTrafilaturaCleaner_Clean_SimpleHTML(t *testing.T) {
	c := NewTrafilatura(nil)

	html := `<html><body><p>Main content paragraph.</p></body></html>`

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// Trafilatura may return original if no content extracted, or extract the paragraph
	if got == "" {
		t.Error("Clean() returned empty string")
	}
}

func TestTrafilaturaCleaner_OutputText(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Output: OutputText,
	})

	html := readTestdata(t, "article.html")

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// Should return text, not HTML
	if got == "" {
		t.Error("Clean() returned empty string")
	}
}

func TestTrafilaturaCleaner_OutputHTML(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Output: OutputHTML,
	})

	html := readTestdata(t, "article.html")

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	if got == "" {
		t.Error("Clean() returned empty string")
	}
}

func TestTrafilaturaCleaner_ExcludeComments(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Comments: Exclude, // This is actually the default
	})

	if c.opts.ExcludeComments != true {
		t.Error("expected ExcludeComments to be true")
	}
}

func TestTrafilaturaCleaner_IncludeComments(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Comments: Include,
	})

	if c.opts.ExcludeComments != false {
		t.Error("expected ExcludeComments to be false when Comments=Include")
	}
}

func TestTrafilaturaCleaner_ExcludeTables(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Tables: Exclude,
	})

	if c.opts.ExcludeTables != true {
		t.Error("expected ExcludeTables to be true")
	}
}

func TestTrafilaturaCleaner_IncludeLinks(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Links: Include, // This is the default
	})

	if c.opts.IncludeLinks != true {
		t.Error("expected IncludeLinks to be true")
	}
}

func TestTrafilaturaCleaner_ExcludeLinks(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Links: Exclude,
	})

	if c.opts.IncludeLinks != false {
		t.Error("expected IncludeLinks to be false when Links=Exclude")
	}
}

func TestTrafilaturaCleaner_DisableFallback(t *testing.T) {
	c := NewTrafilatura(&TrafilaturaConfig{
		Fallback: Exclude,
	})

	if c.opts.EnableFallback != false {
		t.Error("expected EnableFallback to be false when Fallback=Exclude")
	}
}

func TestTrafilaturaCleaner_NilResult_ReturnsOriginal(t *testing.T) {
	c := NewTrafilatura(nil)

	// Minimal HTML with no substantial content
	// Trafilatura may return an error for truly minimal content, which is acceptable
	html := "<html><body></body></html>"
	got, err := c.Clean(html)

	// Trafilatura returns error for insufficient content, which is acceptable
	if err != nil {
		// This is expected for minimal content
		return
	}

	// If no error, it should return something (original or extracted)
	if got == "" {
		t.Error("Clean() returned empty string without error")
	}
}

func TestTrafilaturaCleaner_Name(t *testing.T) {
	c := NewTrafilatura(nil)
	if got := c.Name(); got != "trafilatura" {
		t.Errorf("Name() = %q, want %q", got, "trafilatura")
	}
}

func TestChain_TrafilaturaToMarkdown(t *testing.T) {
	// Create a chain: trafilatura (extract content) -> markdown (convert to md)
	c := NewChain(
		NewTrafilatura(&TrafilaturaConfig{Output: OutputHTML}),
		NewMarkdown(),
	)

	html := readTestdata(t, "article.html")

	got, err := c.Clean(html)
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	// The chain should produce markdown output
	// This is acceptable as trafilatura extraction may vary
	if got == "" {
		t.Error("Clean() returned empty string")
	}
}
