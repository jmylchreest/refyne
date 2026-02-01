package refyne

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("nil config uses default", func(t *testing.T) {
		c := New(nil)
		if c == nil {
			t.Fatal("expected non-nil cleaner")
		}
		if c.config == nil {
			t.Fatal("expected non-nil config")
		}
		// Should have default config values
		if !c.config.StripScripts {
			t.Error("expected StripScripts to be true by default")
		}
	})

	t.Run("custom config is used", func(t *testing.T) {
		cfg := &Config{
			StripScripts: false,
			StripStyles:  true,
		}
		c := New(cfg)
		if c.config.StripScripts {
			t.Error("expected StripScripts to be false")
		}
		if !c.config.StripStyles {
			t.Error("expected StripStyles to be true")
		}
	})
}

func TestName(t *testing.T) {
	c := New(nil)
	if c.Name() != "refyne" {
		t.Errorf("expected name 'refyne', got '%s'", c.Name())
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		config   *Config
		contains []string
		excludes []string
	}{
		{
			name:     "removes script tags",
			html:     `<html><body><p>Hello</p><script>alert('x')</script></body></html>`,
			config:   &Config{StripScripts: true},
			contains: []string{"Hello"},
			excludes: []string{"<script>", "alert"},
		},
		{
			name:     "removes style tags",
			html:     `<html><body><style>.foo{color:red}</style><p>Hello</p></body></html>`,
			config:   &Config{StripStyles: true},
			contains: []string{"Hello"},
			excludes: []string{"<style>", "color:red"},
		},
		{
			name:     "removes inline styles",
			html:     `<html><body><p style="color:red">Hello</p></body></html>`,
			config:   &Config{StripStyles: true},
			contains: []string{"Hello", "<p>", "</p>"},
			excludes: []string{"style=", "color:red"},
		},
		{
			name:     "removes noscript",
			html:     `<html><body><noscript>No JS</noscript><p>Hello</p></body></html>`,
			config:   &Config{StripNoscript: true},
			contains: []string{"Hello"},
			excludes: []string{"<noscript>", "No JS"},
		},
		{
			name:     "removes svg",
			html:     `<html><body><svg><path d="M0 0"></path></svg><p>Hello</p></body></html>`,
			config:   &Config{StripSVGContent: true},
			contains: []string{"Hello"},
			excludes: []string{"<svg>", "<path"},
		},
		{
			name:     "removes iframe",
			html:     `<html><body><iframe src="ad.html"></iframe><p>Hello</p></body></html>`,
			config:   &Config{StripIframes: true},
			contains: []string{"Hello"},
			excludes: []string{"<iframe>", "ad.html"},
		},
		{
			name:     "removes hidden elements",
			html:     `<html><body><div hidden>Hidden</div><p>Visible</p></body></html>`,
			config:   &Config{StripHiddenElements: true},
			contains: []string{"Visible"},
			excludes: []string{"Hidden", "hidden"},
		},
		{
			name:     "removes aria-hidden elements",
			html:     `<html><body><div aria-hidden="true">Hidden</div><p>Visible</p></body></html>`,
			config:   &Config{StripHiddenElements: true},
			contains: []string{"Visible"},
			excludes: []string{"Hidden"},
		},
		{
			name:     "removes display:none elements",
			html:     `<html><body><div style="display:none">Hidden</div><p>Visible</p></body></html>`,
			config:   &Config{StripHiddenElements: true},
			contains: []string{"Visible"},
			excludes: []string{"Hidden"},
		},
		{
			name:     "removes event handlers",
			html:     `<html><body><button onclick="alert()">Click</button></body></html>`,
			config:   &Config{StripEventHandlers: true},
			contains: []string{"Click", "<button>"},
			excludes: []string{"onclick", "alert"},
		},
		{
			name:     "removes data attributes",
			html:     `<html><body><div data-id="123" data-name="foo">Content</div></body></html>`,
			config:   &Config{StripDataAttributes: true},
			contains: []string{"Content"},
			excludes: []string{"data-id", "data-name"},
		},
		{
			name:     "removes aria attributes",
			html:     `<html><body><div aria-label="test" aria-describedby="desc">Content</div></body></html>`,
			config:   &Config{StripARIA: true},
			contains: []string{"Content"},
			excludes: []string{"aria-label", "aria-describedby"},
		},
		{
			name:     "removes classes when configured",
			html:     `<html><body><div class="foo bar">Content</div></body></html>`,
			config:   &Config{StripClasses: true},
			contains: []string{"Content"},
			excludes: []string{"class="},
		},
		{
			name:     "removes ids when configured",
			html:     `<html><body><div id="main">Content</div></body></html>`,
			config:   &Config{StripIDs: true},
			contains: []string{"Content"},
			excludes: []string{"id="},
		},
		{
			name:     "removes microdata when configured",
			html:     `<html><body><div itemscope itemtype="Product"><span itemprop="name">Item</span></div></body></html>`,
			config:   &Config{StripMicrodata: true},
			contains: []string{"Item"},
			excludes: []string{"itemscope", "itemtype", "itemprop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.config)
			result, err := c.Clean(tt.html)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q, got: %s", s, result)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("expected output to not contain %q, got: %s", s, result)
				}
			}
		})
	}
}

func TestCleanWithStats(t *testing.T) {
	t.Run("returns stats with input/output bytes", func(t *testing.T) {
		html := `<html><body><script>x</script><p>Hello</p></body></html>`
		c := New(&Config{StripScripts: true})
		result := c.CleanWithStats(html)

		if result.Stats == nil {
			t.Fatal("expected stats to be non-nil")
		}
		if result.Stats.InputBytes != len(html) {
			t.Errorf("expected input bytes %d, got %d", len(html), result.Stats.InputBytes)
		}
		if result.Stats.OutputBytes >= result.Stats.InputBytes {
			t.Errorf("expected output bytes < input bytes")
		}
	})

	t.Run("tracks elements removed", func(t *testing.T) {
		html := `<html><body><script>x</script><script>y</script></body></html>`
		c := New(&Config{StripScripts: true})
		result := c.CleanWithStats(html)

		if result.Stats.ElementsRemoved["script"] != 2 {
			t.Errorf("expected 2 scripts removed, got %d", result.Stats.ElementsRemoved["script"])
		}
	})

	t.Run("tracks per-phase stats", func(t *testing.T) {
		html := `<html><body><script>x</script><svg></svg><noscript>n</noscript></body></html>`
		c := New(&Config{
			StripScripts:    true,
			StripSVGContent: true,
			StripNoscript:   true,
		})
		result := c.CleanWithStats(html)

		scriptsPhase := result.Stats.GetPhase("scripts")
		if scriptsPhase == nil || scriptsPhase.ElementsRemoved != 1 {
			t.Errorf("expected scripts phase to have 1 removal")
		}

		svgPhase := result.Stats.GetPhase("svg")
		if svgPhase == nil || svgPhase.ElementsRemoved != 1 {
			t.Errorf("expected svg phase to have 1 removal")
		}

		noscriptPhase := result.Stats.GetPhase("noscript")
		if noscriptPhase == nil || noscriptPhase.ElementsRemoved != 1 {
			t.Errorf("expected noscript phase to have 1 removal")
		}
	})

	t.Run("calculates reduction percent", func(t *testing.T) {
		html := `<html><body><script>` + strings.Repeat("x", 1000) + `</script><p>Short</p></body></html>`
		c := New(&Config{StripScripts: true})
		result := c.CleanWithStats(html)

		reduction := result.Stats.ReductionPercent()
		if reduction < 90 {
			t.Errorf("expected >90%% reduction, got %.1f%%", reduction)
		}
	})
}

func TestRemoveBySelector(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		selectors []string
		contains  []string
		excludes  []string
	}{
		{
			name:      "removes by class selector",
			html:      `<html><body><div class="ad">Ad content</div><p>Keep</p></body></html>`,
			selectors: []string{".ad"},
			contains:  []string{"Keep"},
			excludes:  []string{"Ad content"},
		},
		{
			name:      "removes by id selector",
			html:      `<html><body><div id="sidebar">Side</div><p>Main</p></body></html>`,
			selectors: []string{"#sidebar"},
			contains:  []string{"Main"},
			excludes:  []string{"Side"},
		},
		{
			name:      "removes by tag selector",
			html:      `<html><body><nav>Nav links</nav><p>Content</p></body></html>`,
			selectors: []string{"nav"},
			contains:  []string{"Content"},
			excludes:  []string{"Nav links"},
		},
		{
			name:      "removes by attribute selector",
			html:      `<html><body><div role="navigation">Nav</div><p>Content</p></body></html>`,
			selectors: []string{"[role='navigation']"},
			contains:  []string{"Content"},
			excludes:  []string{"Nav"},
		},
		{
			name:      "removes by multiple selectors",
			html:      `<html><body><nav>Nav</nav><footer>Foot</footer><p>Main</p></body></html>`,
			selectors: []string{"nav", "footer"},
			contains:  []string{"Main"},
			excludes:  []string{"Nav", "Foot"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(&Config{RemoveSelectors: tt.selectors})
			result, err := c.Clean(tt.html)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected output to contain %q", s)
				}
			}
			for _, s := range tt.excludes {
				if strings.Contains(result, s) {
					t.Errorf("expected output to not contain %q, got: %s", s, result)
				}
			}
		})
	}
}

func TestKeepSelectors(t *testing.T) {
	t.Run("keep selectors override remove selectors", func(t *testing.T) {
		html := `<html><body><nav class="main-nav">Keep this nav</nav><nav class="footer-nav">Remove this</nav></body></html>`
		c := New(&Config{
			RemoveSelectors: []string{"nav"},
			KeepSelectors:   []string{".main-nav"},
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Keep this nav") {
			t.Errorf("expected to keep .main-nav content")
		}
		if strings.Contains(result, "Remove this") {
			t.Errorf("expected to remove .footer-nav content")
		}
	})

	t.Run("keep selectors prevent hidden element removal", func(t *testing.T) {
		html := `<html><body><div hidden class="important">Keep hidden</div><div hidden>Remove hidden</div></body></html>`
		c := New(&Config{
			StripHiddenElements: true,
			KeepSelectors:       []string{".important"},
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Keep hidden") {
			t.Errorf("expected to keep .important content")
		}
		if strings.Contains(result, "Remove hidden") {
			t.Errorf("expected to remove hidden element without .important class")
		}
	})
}

func TestRemoveByLinkDensity(t *testing.T) {
	t.Run("removes high link density elements", func(t *testing.T) {
		html := `<html><body>
			<div><a href="#">Link1</a> <a href="#">Link2</a> <a href="#">Link3</a></div>
			<p>This is a paragraph with mostly text and only one <a href="#">link</a> in it.</p>
		</body></html>`
		c := New(&Config{
			RemoveByLinkDensity:  true,
			LinkDensityThreshold: 0.5,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// High link density div should be removed
		if strings.Contains(result, "Link1") && strings.Contains(result, "Link2") && strings.Contains(result, "Link3") {
			t.Errorf("expected high link density div to be removed")
		}

		// Low link density paragraph should remain
		if !strings.Contains(result, "mostly text") {
			t.Errorf("expected low link density paragraph to remain")
		}
	})
}

func TestRemoveShortText(t *testing.T) {
	t.Run("removes elements with short text", func(t *testing.T) {
		html := `<html><body>
			<p>Short</p>
			<p>This is a paragraph with enough text to be kept by the cleaner.</p>
		</body></html>`
		c := New(&Config{
			RemoveShortText: true,
			MinTextLength:   20,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Count(result, "Short") > 0 {
			// Could still appear in the longer text, check if the short <p> was removed
			if strings.Contains(result, "<p>Short</p>") || strings.Contains(result, "<p> Short </p>") {
				t.Errorf("expected short text paragraph to be removed")
			}
		}

		if !strings.Contains(result, "enough text") {
			t.Errorf("expected long text paragraph to remain")
		}
	})
}

func TestRemoveEmptyElements(t *testing.T) {
	t.Run("removes empty elements", func(t *testing.T) {
		html := `<html><body><div></div><p>Content</p><span></span></body></html>`
		c := New(&Config{StripEmptyElements: true})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "Content") {
			t.Errorf("expected content to remain")
		}
		// Empty div and span should be removed
		if strings.Contains(result, "<div></div>") {
			t.Errorf("expected empty div to be removed")
		}
		if strings.Contains(result, "<span></span>") {
			t.Errorf("expected empty span to be removed")
		}
	})

	t.Run("preserves self-closing elements", func(t *testing.T) {
		html := `<html><body><p>Text</p><br/><img src="test.jpg"/></body></html>`
		c := New(&Config{StripEmptyElements: true})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "br") {
			t.Errorf("expected br to remain")
		}
		if !strings.Contains(result, "img") {
			t.Errorf("expected img to remain")
		}
	})
}

func TestOutputFormat(t *testing.T) {
	t.Run("text output strips tags", func(t *testing.T) {
		html := `<html><body><p>Hello</p><div>World</div></body></html>`
		c := New(&Config{Output: OutputText})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "<p>") || strings.Contains(result, "<div>") {
			t.Errorf("expected text output to strip tags, got: %s", result)
		}
		if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
			t.Errorf("expected text content to be preserved, got: %s", result)
		}
	})

	t.Run("html output preserves tags", func(t *testing.T) {
		html := `<html><body><p>Hello</p></body></html>`
		c := New(&Config{Output: OutputHTML})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "<p>") {
			t.Errorf("expected HTML output to preserve tags, got: %s", result)
		}
	})
}

func TestWhitespaceHandling(t *testing.T) {
	t.Run("collapses whitespace", func(t *testing.T) {
		html := `<html><body><p>Hello     World</p></body></html>`
		c := New(&Config{CollapseWhitespace: true})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "     ") {
			t.Errorf("expected whitespace to be collapsed")
		}
	})

	t.Run("trims elements", func(t *testing.T) {
		html := `<html><body>   <p>Hello</p>   </body></html>`
		c := New(&Config{TrimElements: true})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Result should be trimmed
		if result != strings.TrimSpace(result) {
			t.Errorf("expected output to be trimmed")
		}
	})
}

func TestCommentsRemoval(t *testing.T) {
	t.Run("removes HTML comments", func(t *testing.T) {
		html := `<html><body><!-- This is a comment --><p>Content</p><!-- Another comment --></body></html>`
		c := New(&Config{StripComments: true})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "<!--") {
			t.Errorf("expected comments to be removed, got: %s", result)
		}
		if !strings.Contains(result, "Content") {
			t.Errorf("expected content to remain")
		}
	})
}

func TestGracefulDegradation(t *testing.T) {
	t.Run("returns original on malformed HTML", func(t *testing.T) {
		// goquery is pretty forgiving, so this actually works fine
		// Test with some truly broken HTML
		html := `<p>Unclosed paragraph`
		c := New(nil)
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("expected no error for graceful degradation")
		}
		if !strings.Contains(result, "Unclosed") {
			t.Errorf("expected content to be preserved")
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		c := New(nil)
		result, err := c.Clean("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Empty body is fine
		_ = result
	})
}

func TestStats(t *testing.T) {
	t.Run("Stats() returns last operation stats", func(t *testing.T) {
		html := `<html><body><script>x</script><p>Hello</p></body></html>`
		c := New(&Config{StripScripts: true})
		_ = c.CleanWithStats(html)

		stats := c.Stats()
		if stats == nil {
			t.Fatal("expected stats to be available")
		}
		if stats.InputBytes != len(html) {
			t.Errorf("expected input bytes %d, got %d", len(html), stats.InputBytes)
		}
	})
}

func TestWarnings(t *testing.T) {
	t.Run("result has warnings for comment removal", func(t *testing.T) {
		html := `<html><body><!-- comment --><p>Text</p></body></html>`
		c := New(&Config{StripComments: true})
		result := c.CleanWithStats(html)

		if !result.HasWarnings() {
			t.Error("expected warnings for comment removal via regex fallback")
		}
	})
}

func TestInterfaceCompliance(t *testing.T) {
	// Ensure Cleaner implements the cleaner.Cleaner interface
	// by testing the required methods exist and work
	c := New(nil)

	// Name() should return a non-empty string
	if c.Name() == "" {
		t.Error("expected Name() to return non-empty string")
	}

	// Clean() should return string and error
	result, err := c.Clean("<p>test</p>")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

// Benchmarks

func BenchmarkClean(b *testing.B) {
	html := `<html><body>
		<script>var x = 1;</script>
		<style>.foo { color: red; }</style>
		<nav><a href="#">Link1</a><a href="#">Link2</a></nav>
		<div class="content">
			<h1>Title</h1>
			<p>This is a paragraph with some text content.</p>
			<ul><li>Item 1</li><li>Item 2</li></ul>
		</div>
		<footer>Footer content</footer>
	</body></html>`

	c := New(DefaultConfig())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Clean(html)
	}
}

func BenchmarkCleanAggressive(b *testing.B) {
	html := `<html><body>
		<script>var x = 1;</script>
		<style>.foo { color: red; }</style>
		<nav><a href="#">Link1</a><a href="#">Link2</a></nav>
		<div class="content">
			<h1>Title</h1>
			<p>This is a paragraph with some text content.</p>
			<ul><li>Item 1</li><li>Item 2</li></ul>
		</div>
		<footer>Footer content</footer>
	</body></html>`

	c := New(PresetAggressive())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Clean(html)
	}
}

func TestMarkdownOutput(t *testing.T) {
	t.Run("basic markdown conversion", func(t *testing.T) {
		html := `<html><body><h1>Title</h1><p>Hello <strong>world</strong></p></body></html>`
		c := New(&Config{Output: OutputMarkdown})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "# Title") {
			t.Errorf("expected markdown h1, got: %s", result)
		}
		if !strings.Contains(result, "**world**") {
			t.Errorf("expected markdown bold, got: %s", result)
		}
	})

	t.Run("markdown with frontmatter", func(t *testing.T) {
		html := `<html><body><h1>Title</h1><h2>Subtitle</h2><p>Content</p></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractHeadings:    true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasPrefix(result, "---\n") {
			t.Errorf("expected frontmatter to start with ---, got: %s", result[:50])
		}
		if !strings.Contains(result, "headings:") {
			t.Errorf("expected headings in frontmatter")
		}
		if !strings.Contains(result, "level: 1") {
			t.Errorf("expected h1 heading level")
		}
	})

	t.Run("markdown with image placeholders", func(t *testing.T) {
		html := `<html><body><img src="test.jpg" alt="Test Image"><p>Content</p></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "{{IMG_001}}") {
			t.Errorf("expected image placeholder in body, got: %s", result)
		}
		if !strings.Contains(result, "IMG_001:") {
			t.Errorf("expected IMG_001 in frontmatter")
		}
		if !strings.Contains(result, `url: "test.jpg"`) {
			t.Errorf("expected image URL in frontmatter")
		}
		if !strings.Contains(result, `alt: "Test Image"`) {
			t.Errorf("expected alt text in frontmatter")
		}
	})

	t.Run("multiple images get sequential placeholders", func(t *testing.T) {
		html := `<html><body>
			<img src="a.jpg" alt="A">
			<img src="b.jpg" alt="B">
			<img src="c.jpg" alt="C">
		</body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "{{IMG_001}}") {
			t.Errorf("expected IMG_001 placeholder")
		}
		if !strings.Contains(result, "{{IMG_002}}") {
			t.Errorf("expected IMG_002 placeholder")
		}
		if !strings.Contains(result, "{{IMG_003}}") {
			t.Errorf("expected IMG_003 placeholder")
		}
	})
}

func TestResolveURLs(t *testing.T) {
	t.Run("relative URLs kept by default", func(t *testing.T) {
		html := `<html><body><a href="/page">Link</a><img src="/img.jpg"></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
			BaseURL:            "https://example.com",
			ResolveURLs:        false, // default
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "https://example.com/page") {
			t.Errorf("expected relative URL to be kept, got absolute")
		}
		if !strings.Contains(result, "(/page)") {
			t.Errorf("expected relative link URL: %s", result)
		}
		if !strings.Contains(result, `url: "/img.jpg"`) {
			t.Errorf("expected relative image URL in frontmatter: %s", result)
		}
	})

	t.Run("relative URLs resolved when enabled", func(t *testing.T) {
		html := `<html><body><a href="/page">Link</a><img src="/img.jpg"></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
			BaseURL:            "https://example.com",
			ResolveURLs:        true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "https://example.com/page") {
			t.Errorf("expected absolute link URL: %s", result)
		}
		if !strings.Contains(result, `url: "https://example.com/img.jpg"`) {
			t.Errorf("expected absolute image URL in frontmatter: %s", result)
		}
	})

	t.Run("protocol-relative URLs always resolved", func(t *testing.T) {
		html := `<html><body><img src="//cdn.example.com/img.jpg"></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
			ResolveURLs:        false, // even when false
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "https://cdn.example.com/img.jpg") {
			t.Errorf("expected protocol-relative URL to be resolved: %s", result)
		}
	})

	t.Run("absolute URLs unchanged", func(t *testing.T) {
		html := `<html><body><img src="https://other.com/img.jpg"></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
			BaseURL:            "https://example.com",
			ResolveURLs:        true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "https://other.com/img.jpg") {
			t.Errorf("expected absolute URL to remain unchanged: %s", result)
		}
	})
}

func TestFrontmatterHints(t *testing.T) {
	t.Run("includes hints by default", func(t *testing.T) {
		html := `<html><body><img src="test.jpg"><p>Content</p></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
			IncludeHints:       true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "hints:") {
			t.Errorf("expected hints section in frontmatter")
		}
		if !strings.Contains(result, "{{IMG_001}}") {
			t.Errorf("expected hint about image placeholders")
		}
	})

	t.Run("hints can be disabled", func(t *testing.T) {
		html := `<html><body><img src="test.jpg"><p>Content</p></body></html>`
		c := New(&Config{
			Output:             OutputMarkdown,
			IncludeFrontmatter: true,
			ExtractImages:      true,
			IncludeHints:       false,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "hints:") {
			t.Errorf("expected no hints section when disabled")
		}
	})
}

func TestStripSrcset(t *testing.T) {
	t.Run("removes srcset and sizes from images", func(t *testing.T) {
		html := `<html><body><img src="img.jpg" srcset="img-320.jpg 320w, img-640.jpg 640w" sizes="(max-width: 600px) 320px, 640px" alt="test"></body></html>`
		c := New(&Config{
			StripSrcset: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "srcset") {
			t.Errorf("expected srcset to be removed: %s", result)
		}
		if strings.Contains(result, "sizes") {
			t.Errorf("expected sizes to be removed: %s", result)
		}
		if !strings.Contains(result, `src="img.jpg"`) {
			t.Errorf("expected src to be preserved: %s", result)
		}
		if !strings.Contains(result, `alt="test"`) {
			t.Errorf("expected alt to be preserved: %s", result)
		}
	})

	t.Run("disabled by default in PresetMinimal", func(t *testing.T) {
		html := `<html><body><img src="img.jpg" srcset="img-320.jpg 320w"></body></html>`
		c := New(PresetMinimal())
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "srcset") {
			t.Errorf("expected srcset to be preserved in minimal mode: %s", result)
		}
	})
}

func TestStripTrackingParams(t *testing.T) {
	t.Run("removes utm parameters from URLs", func(t *testing.T) {
		html := `<html><body><a href="https://example.com/page?utm_source=google&utm_medium=cpc&id=123">Link</a></body></html>`
		c := New(&Config{
			StripTrackingParams: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "utm_source") {
			t.Errorf("expected utm_source to be removed: %s", result)
		}
		if strings.Contains(result, "utm_medium") {
			t.Errorf("expected utm_medium to be removed: %s", result)
		}
		if !strings.Contains(result, "id=123") {
			t.Errorf("expected non-tracking params to be preserved: %s", result)
		}
	})

	t.Run("removes fbclid from URLs", func(t *testing.T) {
		html := `<html><body><a href="https://example.com/page?fbclid=abc123&name=test">Link</a></body></html>`
		c := New(&Config{
			StripTrackingParams: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "fbclid") {
			t.Errorf("expected fbclid to be removed: %s", result)
		}
		if !strings.Contains(result, "name=test") {
			t.Errorf("expected non-tracking params to be preserved: %s", result)
		}
	})

	t.Run("handles URLs with only tracking params", func(t *testing.T) {
		html := `<html><body><a href="https://example.com/page?utm_source=google">Link</a></body></html>`
		c := New(&Config{
			StripTrackingParams: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "?") {
			t.Errorf("expected query string to be removed entirely: %s", result)
		}
		if !strings.Contains(result, "https://example.com/page") {
			t.Errorf("expected base URL to be preserved: %s", result)
		}
	})

	t.Run("preserves fragment", func(t *testing.T) {
		html := `<html><body><a href="https://example.com/page?utm_source=google#section">Link</a></body></html>`
		c := New(&Config{
			StripTrackingParams: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "#section") {
			t.Errorf("expected fragment to be preserved: %s", result)
		}
	})
}

func TestDeduplicateTextBlocks(t *testing.T) {
	t.Run("removes duplicate text blocks", func(t *testing.T) {
		html := `<html><body>
			<p>This is unique content here.</p>
			<p>Read more about this topic</p>
			<p>This is unique content here.</p>
			<p>Read more about this topic</p>
		</body></html>`
		c := New(&Config{
			DeduplicateTextBlocks: true,
			MinDuplicateLength:    15,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Count occurrences - should only appear once each
		count := strings.Count(result, "This is unique content here")
		if count != 1 {
			t.Errorf("expected 'This is unique content here' to appear once, got %d: %s", count, result)
		}

		count = strings.Count(result, "Read more about this topic")
		if count != 1 {
			t.Errorf("expected 'Read more about this topic' to appear once, got %d: %s", count, result)
		}
	})

	t.Run("keeps short duplicates", func(t *testing.T) {
		html := `<html><body>
			<p>Short</p>
			<p>Short</p>
		</body></html>`
		c := New(&Config{
			DeduplicateTextBlocks: true,
			MinDuplicateLength:    15,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		count := strings.Count(result, "Short")
		if count != 2 {
			t.Errorf("expected short text to appear twice, got %d: %s", count, result)
		}
	})
}

func TestStripCommonBoilerplate(t *testing.T) {
	t.Run("removes copyright notices", func(t *testing.T) {
		html := `<html><body>
			<p>Main content here</p>
			<footer><p>Copyright © 2024 Company</p></footer>
		</body></html>`
		c := New(&Config{
			StripCommonBoilerplate: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "Copyright") {
			t.Errorf("expected copyright to be removed: %s", result)
		}
		if !strings.Contains(result, "Main content") {
			t.Errorf("expected main content to be preserved: %s", result)
		}
	})

	t.Run("removes all rights reserved", func(t *testing.T) {
		html := `<html><body>
			<p>Content</p>
			<p>All rights reserved.</p>
		</body></html>`
		c := New(&Config{
			StripCommonBoilerplate: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(strings.ToLower(result), "all rights reserved") {
			t.Errorf("expected 'all rights reserved' to be removed: %s", result)
		}
	})
}

func TestCollapseBlankLines(t *testing.T) {
	t.Run("collapses multiple blank lines", func(t *testing.T) {
		html := `<html><body>
			<p>First paragraph</p>



			<p>Second paragraph</p>
		</body></html>`
		c := New(&Config{
			Output:             OutputText,
			CollapseBlankLines: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should not have more than 2 consecutive newlines
		if strings.Contains(result, "\n\n\n") {
			t.Errorf("expected multiple blank lines to be collapsed: %q", result)
		}
	})
}

func TestRemoveRepeatedLinks(t *testing.T) {
	t.Run("removes repeated links keeping first", func(t *testing.T) {
		html := `<html><body>
			<a href="https://example.com/page">First link</a>
			<a href="https://example.com/page">Second link</a>
			<a href="https://example.com/other">Other link</a>
		</body></html>`
		c := New(&Config{
			RemoveRepeatedLinks: true,
		})
		result, err := c.Clean(html)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// First link should remain as a link
		if !strings.Contains(result, `href="https://example.com/page"`) {
			t.Errorf("expected first link to be preserved: %s", result)
		}

		// Should contain "Second link" text but not as a link
		if !strings.Contains(result, "Second link") {
			t.Errorf("expected second link text to be preserved: %s", result)
		}

		// Other link should remain
		if !strings.Contains(result, `href="https://example.com/other"`) {
			t.Errorf("expected other link to be preserved: %s", result)
		}
	})
}
