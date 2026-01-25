//go:build markdown

package cleaner

import (
	"strings"
	"testing"
)

func TestRichMarkdownCleaner_Basic(t *testing.T) {
	html := `<html>
	<head><title>Test</title></head>
	<body>
		<h1>Main Title</h1>
		<p>Introduction text.</p>
		<img src="https://example.com/image1.jpg" alt="First image">
		<h2>Step 1: Getting Started</h2>
		<p>Some instructions here.</p>
		<img src="https://example.com/image2.jpg" alt="Step 1 image">
		<a href="https://example.com/link">A link</a>
	</body>
	</html>`

	cleaner := NewRichMarkdown()
	result, err := cleaner.Clean(html)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Should have frontmatter
	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected frontmatter to start with ---")
	}

	// Should contain images section
	if !strings.Contains(result, "images:") {
		t.Error("expected images section in frontmatter")
	}

	// Should contain image URLs
	if !strings.Contains(result, "https://example.com/image1.jpg") {
		t.Error("expected first image URL in frontmatter")
	}
	if !strings.Contains(result, "https://example.com/image2.jpg") {
		t.Error("expected second image URL in frontmatter")
	}

	// Should contain headings section
	if !strings.Contains(result, "headings:") {
		t.Error("expected headings section in frontmatter")
	}

	// Should contain hints
	if !strings.Contains(result, "hints:") {
		t.Error("expected hints section in frontmatter")
	}

	// Should contain markdown content after frontmatter
	if !strings.Contains(result, "# Main Title") {
		t.Error("expected markdown heading in content")
	}
}

func TestRichMarkdownCleaner_LazyLoadedImages(t *testing.T) {
	// Simulates Instructables lazy-loading pattern
	html := `<html>
	<body>
		<h1>Tutorial</h1>
		<img class="lazyload" data-src="https://content.instructables.com/real-image.jpg" src="/assets/img/pixel.png" alt="Lazy image">
		<noscript>
			<img src="https://content.instructables.com/fallback-image.jpg" alt="Fallback">
		</noscript>
	</body>
	</html>`

	cleaner := NewRichMarkdown()
	metadata, err := cleaner.ExtractMetadataOnly(html)
	if err != nil {
		t.Fatalf("ExtractMetadataOnly failed: %v", err)
	}

	// Should extract data-src, not the pixel placeholder
	foundRealImage := false
	foundPixel := false
	for _, img := range metadata.Images {
		if img.URL == "https://content.instructables.com/real-image.jpg" {
			foundRealImage = true
		}
		if strings.Contains(img.URL, "pixel.png") {
			foundPixel = true
		}
	}

	if !foundRealImage {
		t.Error("expected data-src image URL to be extracted")
	}
	if foundPixel {
		t.Error("should not include pixel placeholder in extracted images")
	}
}

func TestRichMarkdownCleaner_ImageContext(t *testing.T) {
	html := `<html>
	<body>
		<h2>Introduction</h2>
		<img src="https://example.com/intro.jpg" alt="Intro">
		<h2>Step 1: Prepare</h2>
		<img src="https://example.com/step1.jpg" alt="Step 1">
		<h2>Step 2: Build</h2>
		<img src="https://example.com/step2.jpg" alt="Step 2">
	</body>
	</html>`

	cleaner := NewRichMarkdown()
	result, err := cleaner.Clean(html)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Check that images have context from their preceding headings
	// The context should be included in the frontmatter
	if !strings.Contains(result, `context: "Introduction"`) {
		t.Error("expected intro image to have Introduction context")
	}
	if !strings.Contains(result, `context: "Step 1: Prepare"`) {
		t.Error("expected step1 image to have Step 1 context")
	}
	if !strings.Contains(result, `context: "Step 2: Build"`) {
		t.Error("expected step2 image to have Step 2 context")
	}
}

func TestRichMarkdownCleaner_NoFrontmatter(t *testing.T) {
	html := `<html><body><h1>Title</h1><p>Content</p></body></html>`

	cleaner := NewRichMarkdown(WithFrontmatter(false))
	result, err := cleaner.Clean(html)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Should NOT have frontmatter
	if strings.HasPrefix(result, "---") {
		t.Error("expected no frontmatter when disabled")
	}

	// Should still have markdown content
	if !strings.Contains(result, "# Title") {
		t.Error("expected markdown content")
	}
}

func TestRichMarkdownCleaner_CustomHints(t *testing.T) {
	html := `<html><body><h1>Title</h1><img src="test.jpg"></body></html>`

	cleaner := NewRichMarkdown(
		WithCustomHints("Custom hint 1", "Custom hint 2"),
	)
	result, err := cleaner.Clean(html)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	if !strings.Contains(result, "Custom hint 1") {
		t.Error("expected custom hint 1")
	}
	if !strings.Contains(result, "Custom hint 2") {
		t.Error("expected custom hint 2")
	}
}

func TestRichMarkdownCleaner_ProtocolRelativeURLs(t *testing.T) {
	html := `<html><body>
		<img src="//example.com/image.jpg" alt="Protocol relative">
	</body></html>`

	cleaner := NewRichMarkdown()
	result, err := cleaner.Clean(html)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	// Should resolve protocol-relative URLs to https
	if !strings.Contains(result, "https://example.com/image.jpg") {
		t.Error("expected protocol-relative URL to be resolved to https")
	}
}

func TestRichMarkdownCleaner_ExtractMetadataOnly(t *testing.T) {
	html := `<html>
	<body>
		<h1>Title</h1>
		<img src="https://example.com/image.jpg" alt="Test">
		<a href="link1">Link 1</a>
		<a href="link2">Link 2</a>
	</body>
	</html>`

	cleaner := NewRichMarkdown()
	metadata, err := cleaner.ExtractMetadataOnly(html)
	if err != nil {
		t.Fatalf("ExtractMetadataOnly failed: %v", err)
	}

	if len(metadata.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(metadata.Images))
	}
	if metadata.Images[0].URL != "https://example.com/image.jpg" {
		t.Errorf("expected image URL, got %s", metadata.Images[0].URL)
	}
	if len(metadata.Headings) != 1 {
		t.Errorf("expected 1 heading, got %d", len(metadata.Headings))
	}
	if metadata.LinksCount != 2 {
		t.Errorf("expected 2 links, got %d", metadata.LinksCount)
	}
}
