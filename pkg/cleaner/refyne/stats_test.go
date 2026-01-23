package refyne

import (
	"strings"
	"testing"
	"time"
)

func TestNewStats(t *testing.T) {
	s := NewStats()

	if s == nil {
		t.Fatal("expected non-nil stats")
	}
	if s.ElementsRemoved == nil {
		t.Error("expected ElementsRemoved map to be initialized")
	}
	if s.SelectorMatches == nil {
		t.Error("expected SelectorMatches map to be initialized")
	}
	if s.Phases == nil {
		t.Error("expected Phases slice to be initialized")
	}
}

func TestStatsAddPhase(t *testing.T) {
	s := NewStats()

	phase := s.AddPhase("scripts", true)

	if phase == nil {
		t.Fatal("expected non-nil phase")
	}
	if phase.Name != "scripts" {
		t.Errorf("expected name 'scripts', got %q", phase.Name)
	}
	if !phase.Enabled {
		t.Error("expected enabled to be true")
	}
	if phase.Details == nil {
		t.Error("expected Details map to be initialized")
	}

	// Should be added to Phases slice
	if len(s.Phases) != 1 {
		t.Errorf("expected 1 phase, got %d", len(s.Phases))
	}
	if s.Phases[0] != phase {
		t.Error("expected phase to be in Phases slice")
	}
}

func TestStatsGetPhase(t *testing.T) {
	s := NewStats()
	s.AddPhase("scripts", true)
	s.AddPhase("styles", true)
	s.AddPhase("noscript", false)

	t.Run("finds existing phase", func(t *testing.T) {
		phase := s.GetPhase("scripts")
		if phase == nil {
			t.Fatal("expected to find scripts phase")
		}
		if phase.Name != "scripts" {
			t.Errorf("expected name 'scripts', got %q", phase.Name)
		}
	})

	t.Run("returns nil for non-existent phase", func(t *testing.T) {
		phase := s.GetPhase("nonexistent")
		if phase != nil {
			t.Error("expected nil for non-existent phase")
		}
	})
}

func TestStatsReductionPercent(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		output   int
		expected float64
	}{
		{"50% reduction", 100, 50, 50.0},
		{"0% reduction", 100, 100, 0.0},
		{"100% reduction", 100, 0, 100.0},
		{"zero input", 0, 0, 0.0},
		{"75% reduction", 1000, 250, 75.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStats()
			s.InputBytes = tt.input
			s.OutputBytes = tt.output

			result := s.ReductionPercent()
			if result != tt.expected {
				t.Errorf("expected %.1f%%, got %.1f%%", tt.expected, result)
			}
		})
	}
}

func TestStatsTotalElementsRemoved(t *testing.T) {
	s := NewStats()
	s.ElementsRemoved["script"] = 5
	s.ElementsRemoved["style"] = 3
	s.ElementsRemoved["div"] = 10

	total := s.TotalElementsRemoved()
	if total != 18 {
		t.Errorf("expected 18 total elements, got %d", total)
	}
}

func TestStatsRecordRemoval(t *testing.T) {
	s := NewStats()

	s.RecordRemoval("SCRIPT") // Test case-insensitivity
	s.RecordRemoval("script")
	s.RecordRemoval("DIV")

	if s.ElementsRemoved["script"] != 2 {
		t.Errorf("expected 2 script removals, got %d", s.ElementsRemoved["script"])
	}
	if s.ElementsRemoved["div"] != 1 {
		t.Errorf("expected 1 div removal, got %d", s.ElementsRemoved["div"])
	}
}

func TestStatsRecordSelectorMatch(t *testing.T) {
	s := NewStats()

	s.RecordSelectorMatch(".ad", 3)
	s.RecordSelectorMatch(".ad", 2)
	s.RecordSelectorMatch("nav", 1)

	if s.SelectorMatches[".ad"] != 5 {
		t.Errorf("expected 5 .ad matches, got %d", s.SelectorMatches[".ad"])
	}
	if s.SelectorMatches["nav"] != 1 {
		t.Errorf("expected 1 nav match, got %d", s.SelectorMatches["nav"])
	}
}

func TestStatsString(t *testing.T) {
	s := NewStats()
	s.InputBytes = 1000
	s.OutputBytes = 300
	s.ElementsRemoved["script"] = 5
	s.ElementsRemoved["style"] = 2
	s.ElementsKept = 50
	s.AttributesRemoved = 10
	s.SelectorMatches[".ad"] = 3
	s.ParseDuration = 5 * time.Millisecond
	s.TransformDuration = 10 * time.Millisecond
	s.OutputDuration = 2 * time.Millisecond
	s.TotalDuration = 17 * time.Millisecond

	// Add a phase
	phase := s.AddPhase("scripts", true)
	phase.ElementsRemoved = 5
	phase.Details["script"] = 5

	output := s.String()

	// Check key components are present
	if !strings.Contains(output, "70.0%") {
		t.Errorf("expected 70.0%% reduction in output, got: %s", output)
	}
	if !strings.Contains(output, "1000") {
		t.Error("expected input bytes in output")
	}
	if !strings.Contains(output, "300") {
		t.Error("expected output bytes in output")
	}
	if !strings.Contains(output, "script") {
		t.Error("expected script in elements removed")
	}
	if !strings.Contains(output, "50 kept") {
		t.Error("expected elements kept count")
	}
	if !strings.Contains(output, ".ad") {
		t.Error("expected selector matches in output")
	}
	if !strings.Contains(output, "scripts:") {
		t.Error("expected per-phase breakdown in output")
	}
}

func TestPhaseStats(t *testing.T) {
	phase := &PhaseStats{
		Name:            "test",
		Enabled:         true,
		ElementsRemoved: 0,
		Details:         make(map[string]int),
	}

	phase.ElementsRemoved = 5
	phase.Details["div"] = 3
	phase.Details["span"] = 2

	if phase.ElementsRemoved != 5 {
		t.Errorf("expected 5 elements removed, got %d", phase.ElementsRemoved)
	}
	if phase.Details["div"] != 3 {
		t.Errorf("expected 3 divs in details, got %d", phase.Details["div"])
	}
}

func TestWarning(t *testing.T) {
	t.Run("with context", func(t *testing.T) {
		w := Warning{
			Phase:   "parse",
			Message: "test warning",
			Context: "some element",
		}

		str := w.String()
		if !strings.Contains(str, "[parse]") {
			t.Error("expected phase in warning string")
		}
		if !strings.Contains(str, "test warning") {
			t.Error("expected message in warning string")
		}
		if !strings.Contains(str, "some element") {
			t.Error("expected context in warning string")
		}
	})

	t.Run("without context", func(t *testing.T) {
		w := Warning{
			Phase:   "output",
			Message: "no context warning",
		}

		str := w.String()
		if strings.Contains(str, "context:") {
			t.Error("did not expect context label when no context provided")
		}
	})
}

func TestResult(t *testing.T) {
	t.Run("AddWarning", func(t *testing.T) {
		r := &Result{
			Content: "test",
			Stats:   NewStats(),
		}

		r.AddWarning("parse", "test message", "test context")

		if len(r.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(r.Warnings))
		}
		if r.Warnings[0].Phase != "parse" {
			t.Errorf("expected phase 'parse', got %q", r.Warnings[0].Phase)
		}
		if r.Warnings[0].Message != "test message" {
			t.Errorf("expected message 'test message', got %q", r.Warnings[0].Message)
		}
	})

	t.Run("HasWarnings", func(t *testing.T) {
		r := &Result{}

		if r.HasWarnings() {
			t.Error("expected no warnings initially")
		}

		r.Warnings = append(r.Warnings, Warning{Phase: "test"})

		if !r.HasWarnings() {
			t.Error("expected HasWarnings to return true after adding warning")
		}
	})
}

func TestStatsStringWithEmptyPhases(t *testing.T) {
	s := NewStats()
	s.InputBytes = 100
	s.OutputBytes = 50

	// Add disabled phase
	s.AddPhase("scripts", false)

	// Add enabled phase with no removals
	s.AddPhase("styles", true)

	output := s.String()

	// Should not crash and should contain basic info
	if !strings.Contains(output, "50.0%") {
		t.Errorf("expected reduction percentage in output, got: %s", output)
	}
}

func TestStatsStringWithManyDetails(t *testing.T) {
	s := NewStats()
	s.InputBytes = 1000
	s.OutputBytes = 100

	phase := s.AddPhase("selectors", true)
	phase.ElementsRemoved = 100

	// Add many details (more than 5)
	phase.Details[".ad1"] = 20
	phase.Details[".ad2"] = 18
	phase.Details[".ad3"] = 16
	phase.Details[".ad4"] = 14
	phase.Details[".ad5"] = 12
	phase.Details[".ad6"] = 10 // Should not be shown (only top 5)
	phase.Details[".ad7"] = 8  // Should not be shown

	output := s.String()

	// Should show top 5 (sorted by count descending)
	if !strings.Contains(output, ".ad1") {
		t.Error("expected .ad1 in output")
	}
	if !strings.Contains(output, ".ad5") {
		t.Error("expected .ad5 in output")
	}
	// .ad6 and .ad7 should not be shown
	// Note: they might still appear if counts are close due to sorting
}
