package refyne

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Test expected defaults
	tests := []struct {
		name     string
		got      bool
		expected bool
	}{
		{"StripScripts", cfg.StripScripts, true},
		{"StripStyles", cfg.StripStyles, true},
		{"StripComments", cfg.StripComments, true},
		{"StripEventHandlers", cfg.StripEventHandlers, true},
		{"StripNoscript", cfg.StripNoscript, false}, // Default: preserve for image fallbacks
		{"StripSVGContent", cfg.StripSVGContent, true},
		{"StripIframes", cfg.StripIframes, true},
		{"StripHiddenElements", cfg.StripHiddenElements, true},
		{"StripDataAttributes", cfg.StripDataAttributes, true},
		{"StripARIA", cfg.StripARIA, true},
		{"StripMicrodata", cfg.StripMicrodata, false}, // Default: preserve for structured data
		{"StripClasses", cfg.StripClasses, false},     // Default: preserve for selectors
		{"StripIDs", cfg.StripIDs, false},             // Default: preserve for anchors
		{"StripEmptyElements", cfg.StripEmptyElements, false},
		{"PreserveLinks", cfg.PreserveLinks, true},
		{"PreserveImages", cfg.PreserveImages, true},
		{"PreserveTables", cfg.PreserveTables, true},
		{"PreserveForms", cfg.PreserveForms, true},
		{"PreserveLists", cfg.PreserveLists, true},
		{"PreserveSemanticTags", cfg.PreserveSemanticTags, true},
		{"RemoveByLinkDensity", cfg.RemoveByLinkDensity, false}, // Off by default
		{"RemoveShortText", cfg.RemoveShortText, false},         // Off by default
		{"CollapseWhitespace", cfg.CollapseWhitespace, true},
		{"TrimElements", cfg.TrimElements, true},
		{"Debug", cfg.Debug, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, tt.got)
			}
		})
	}

	// Test numeric defaults
	if cfg.LinkDensityThreshold != 0.5 {
		t.Errorf("expected LinkDensityThreshold 0.5, got %f", cfg.LinkDensityThreshold)
	}
	if cfg.MinTextLength != 20 {
		t.Errorf("expected MinTextLength 20, got %d", cfg.MinTextLength)
	}
	if cfg.Output != OutputHTML {
		t.Errorf("expected Output HTML, got %s", cfg.Output)
	}

	// Should have default remove selectors
	if len(cfg.RemoveSelectors) == 0 {
		t.Error("expected default RemoveSelectors to be non-empty")
	}
}

func TestPresetMinimal(t *testing.T) {
	cfg := PresetMinimal()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Minimal should only strip scripts, styles, and comments
	if !cfg.StripScripts {
		t.Error("expected StripScripts to be true")
	}
	if !cfg.StripStyles {
		t.Error("expected StripStyles to be true")
	}
	if !cfg.StripComments {
		t.Error("expected StripComments to be true")
	}
	if !cfg.CollapseWhitespace {
		t.Error("expected CollapseWhitespace to be true")
	}

	// Should NOT have aggressive features enabled
	if cfg.StripHiddenElements {
		t.Error("minimal preset should not strip hidden elements")
	}
	if cfg.StripNoscript {
		t.Error("minimal preset should not strip noscript")
	}
	if cfg.StripSVGContent {
		t.Error("minimal preset should not strip SVG")
	}
	if cfg.RemoveByLinkDensity {
		t.Error("minimal preset should not remove by link density")
	}
	if cfg.RemoveShortText {
		t.Error("minimal preset should not remove short text")
	}
	if cfg.StripEmptyElements {
		t.Error("minimal preset should not strip empty elements")
	}

	// Should have no remove selectors
	if len(cfg.RemoveSelectors) > 0 {
		t.Error("minimal preset should have no RemoveSelectors")
	}
}

func TestPresetAggressive(t *testing.T) {
	cfg := PresetAggressive()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Should inherit from default
	if !cfg.StripScripts {
		t.Error("expected StripScripts to be true")
	}
	if !cfg.StripStyles {
		t.Error("expected StripStyles to be true")
	}

	// Aggressive-specific features
	if !cfg.RemoveByLinkDensity {
		t.Error("expected RemoveByLinkDensity to be true")
	}
	if cfg.LinkDensityThreshold != 0.5 {
		t.Errorf("expected LinkDensityThreshold 0.5, got %f", cfg.LinkDensityThreshold)
	}
	if !cfg.RemoveShortText {
		t.Error("expected RemoveShortText to be true")
	}
	if cfg.MinTextLength != 25 {
		t.Errorf("expected MinTextLength 25, got %d", cfg.MinTextLength)
	}
	if !cfg.StripEmptyElements {
		t.Error("expected StripEmptyElements to be true")
	}

	// Should have additional nav/header/footer selectors
	foundNav := false
	foundHeader := false
	foundFooter := false
	for _, s := range cfg.RemoveSelectors {
		switch s {
		case "nav":
			foundNav = true
		case "header":
			foundHeader = true
		case "footer":
			foundFooter = true
		}
	}
	if !foundNav || !foundHeader || !foundFooter {
		t.Error("expected aggressive preset to include nav, header, footer in RemoveSelectors")
	}
}

func TestConfigMerge(t *testing.T) {
	t.Run("nil other returns original", func(t *testing.T) {
		base := DefaultConfig()
		merged := base.Merge(nil)
		if merged != base {
			t.Error("expected same config returned when merging nil")
		}
	})

	t.Run("other values override base", func(t *testing.T) {
		base := &Config{
			StripScripts: false,
			StripStyles:  false,
		}
		other := &Config{
			StripScripts: true,
		}
		merged := base.Merge(other)

		if !merged.StripScripts {
			t.Error("expected StripScripts to be overridden to true")
		}
		if merged.StripStyles {
			t.Error("expected StripStyles to remain false")
		}
	})

	t.Run("selectors are appended not replaced", func(t *testing.T) {
		base := &Config{
			RemoveSelectors: []string{".ad"},
		}
		other := &Config{
			RemoveSelectors: []string{".banner", "nav"},
		}
		merged := base.Merge(other)

		if len(merged.RemoveSelectors) != 3 {
			t.Errorf("expected 3 selectors, got %d", len(merged.RemoveSelectors))
		}

		found := map[string]bool{}
		for _, s := range merged.RemoveSelectors {
			found[s] = true
		}
		if !found[".ad"] || !found[".banner"] || !found["nav"] {
			t.Error("expected all selectors to be present")
		}
	})

	t.Run("duplicate selectors are not added", func(t *testing.T) {
		base := &Config{
			RemoveSelectors: []string{".ad", "nav"},
		}
		other := &Config{
			RemoveSelectors: []string{".ad", ".banner"}, // .ad is duplicate
		}
		merged := base.Merge(other)

		count := 0
		for _, s := range merged.RemoveSelectors {
			if s == ".ad" {
				count++
			}
		}
		if count != 1 {
			t.Errorf("expected .ad to appear once, got %d times", count)
		}
	})

	t.Run("keep selectors are appended", func(t *testing.T) {
		base := &Config{
			KeepSelectors: []string{".main"},
		}
		other := &Config{
			KeepSelectors: []string{".content"},
		}
		merged := base.Merge(other)

		if len(merged.KeepSelectors) != 2 {
			t.Errorf("expected 2 keep selectors, got %d", len(merged.KeepSelectors))
		}
	})

	t.Run("numeric values override when non-zero", func(t *testing.T) {
		base := &Config{
			LinkDensityThreshold: 0.5,
			MinTextLength:        20,
		}
		other := &Config{
			LinkDensityThreshold: 0.7,
		}
		merged := base.Merge(other)

		if merged.LinkDensityThreshold != 0.7 {
			t.Errorf("expected threshold 0.7, got %f", merged.LinkDensityThreshold)
		}
		if merged.MinTextLength != 20 {
			t.Errorf("expected MinTextLength 20 to be preserved, got %d", merged.MinTextLength)
		}
	})
}

func TestOutputFormatConstants(t *testing.T) {
	tests := []struct {
		format   OutputFormat
		expected string
	}{
		{OutputHTML, "html"},
		{OutputText, "text"},
		{OutputMarkdown, "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.format) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.format)
			}
		})
	}
}

func TestConfigImmutability(t *testing.T) {
	// Ensure that modifying a preset doesn't affect future calls
	cfg1 := DefaultConfig()
	cfg1.StripScripts = false
	cfg1.RemoveSelectors = append(cfg1.RemoveSelectors, ".custom")

	cfg2 := DefaultConfig()
	if !cfg2.StripScripts {
		t.Error("modifying cfg1 should not affect cfg2")
	}

	// Check that the custom selector wasn't added to cfg2
	for _, s := range cfg2.RemoveSelectors {
		if s == ".custom" {
			t.Error("modifying cfg1.RemoveSelectors should not affect cfg2")
		}
	}
}

func TestConfigDefaultSelectorsContent(t *testing.T) {
	cfg := DefaultConfig()

	// Test that specific known selectors are present
	expectedSelectors := []string{
		"[aria-hidden='true']",
		"[hidden]",
		".ad",
		".ads",
		".advertisement",
		"[class*='cookie']",
	}

	selectorSet := make(map[string]bool)
	for _, s := range cfg.RemoveSelectors {
		selectorSet[s] = true
	}

	for _, expected := range expectedSelectors {
		if !selectorSet[expected] {
			t.Errorf("expected selector %q in default config", expected)
		}
	}
}

func TestConfigDeepCopy(t *testing.T) {
	// Merge should create a copy, not modify the original
	original := &Config{
		StripScripts:    true,
		RemoveSelectors: []string{".ad"},
		KeepSelectors:   []string{".main"},
	}

	other := &Config{
		RemoveSelectors: []string{".banner"},
	}

	merged := original.Merge(other)

	// Verify original is unchanged
	if len(original.RemoveSelectors) != 1 || original.RemoveSelectors[0] != ".ad" {
		t.Error("original.RemoveSelectors was modified")
	}

	// Verify merged has both
	if len(merged.RemoveSelectors) != 2 {
		t.Errorf("expected merged to have 2 selectors, got %d", len(merged.RemoveSelectors))
	}
}
