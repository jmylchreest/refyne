package refyne

import (
	"fmt"
	"strings"
	"time"
)

// Stats captures metrics about what the cleaner did.
type Stats struct {
	// Size metrics
	InputBytes  int `json:"input_bytes"`
	OutputBytes int `json:"output_bytes"`

	// Element counts
	ElementsRemoved map[string]int `json:"elements_removed"` // tag -> count
	ElementsKept    int            `json:"elements_kept"`

	// Attribute cleaning
	AttributesRemoved int `json:"attributes_removed"`

	// Selector matches
	SelectorMatches map[string]int `json:"selector_matches"` // selector -> count

	// Heuristic triggers
	LinkDensityRemovals   int `json:"link_density_removals"`
	ShortTextRemovals     int `json:"short_text_removals"`
	HiddenElementRemovals int `json:"hidden_element_removals"`
	EmptyElementRemovals  int `json:"empty_element_removals"`

	// Timing
	ParseDuration     time.Duration `json:"parse_duration_ms"`
	TransformDuration time.Duration `json:"transform_duration_ms"`
	OutputDuration    time.Duration `json:"output_duration_ms"`
	TotalDuration     time.Duration `json:"total_duration_ms"`
}

// NewStats creates a new Stats instance with initialized maps.
func NewStats() *Stats {
	return &Stats{
		ElementsRemoved: make(map[string]int),
		SelectorMatches: make(map[string]int),
	}
}

// ReductionPercent returns the percentage reduction in size.
func (s *Stats) ReductionPercent() float64 {
	if s.InputBytes == 0 {
		return 0
	}
	return float64(s.InputBytes-s.OutputBytes) / float64(s.InputBytes) * 100
}

// TotalElementsRemoved returns the sum of all removed elements.
func (s *Stats) TotalElementsRemoved() int {
	total := 0
	for _, count := range s.ElementsRemoved {
		total += count
	}
	return total
}

// RecordRemoval records that an element was removed.
func (s *Stats) RecordRemoval(tag string) {
	s.ElementsRemoved[strings.ToLower(tag)]++
}

// RecordSelectorMatch records that a selector matched elements.
func (s *Stats) RecordSelectorMatch(selector string, count int) {
	s.SelectorMatches[selector] += count
}

// String returns a human-readable summary of the stats.
func (s *Stats) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Size: %d -> %d bytes (%.1f%% reduction)\n",
		s.InputBytes, s.OutputBytes, s.ReductionPercent()))

	sb.WriteString(fmt.Sprintf("Elements: %d removed, %d kept\n",
		s.TotalElementsRemoved(), s.ElementsKept))

	if len(s.ElementsRemoved) > 0 {
		sb.WriteString("Removed by tag: ")
		parts := make([]string, 0, len(s.ElementsRemoved))
		for tag, count := range s.ElementsRemoved {
			parts = append(parts, fmt.Sprintf("%s=%d", tag, count))
		}
		sb.WriteString(strings.Join(parts, ", "))
		sb.WriteString("\n")
	}

	if s.AttributesRemoved > 0 {
		sb.WriteString(fmt.Sprintf("Attributes removed: %d\n", s.AttributesRemoved))
	}

	if s.LinkDensityRemovals > 0 {
		sb.WriteString(fmt.Sprintf("Link density removals: %d\n", s.LinkDensityRemovals))
	}

	if s.HiddenElementRemovals > 0 {
		sb.WriteString(fmt.Sprintf("Hidden element removals: %d\n", s.HiddenElementRemovals))
	}

	sb.WriteString(fmt.Sprintf("Timing: parse=%v, transform=%v, output=%v, total=%v\n",
		s.ParseDuration.Round(time.Millisecond),
		s.TransformDuration.Round(time.Millisecond),
		s.OutputDuration.Round(time.Millisecond),
		s.TotalDuration.Round(time.Millisecond)))

	return sb.String()
}

// Warning represents a non-fatal issue encountered during cleaning.
type Warning struct {
	Phase   string `json:"phase"`   // "parse", "transform", "output"
	Message string `json:"message"` // Human-readable description
	Context string `json:"context"` // Element or selector that caused issue
}

// String returns a formatted warning message.
func (w Warning) String() string {
	if w.Context != "" {
		return fmt.Sprintf("[%s] %s (context: %s)", w.Phase, w.Message, w.Context)
	}
	return fmt.Sprintf("[%s] %s", w.Phase, w.Message)
}

// Result contains the output of a cleaning operation.
type Result struct {
	// Content is the cleaned output. On parse errors, this contains the original input.
	Content string `json:"content"`

	// Stats contains metrics about what was done.
	Stats *Stats `json:"stats"`

	// Warnings contains non-fatal issues encountered.
	Warnings []Warning `json:"warnings,omitempty"`

	// Error is set only on catastrophic failures (content is still returned).
	Error error `json:"error,omitempty"`
}

// AddWarning adds a warning to the result.
func (r *Result) AddWarning(phase, message, context string) {
	r.Warnings = append(r.Warnings, Warning{
		Phase:   phase,
		Message: message,
		Context: context,
	})
}

// HasWarnings returns true if any warnings were recorded.
func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}
