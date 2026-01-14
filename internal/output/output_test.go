package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Test data structure
type testItem struct {
	Name  string `json:"name" yaml:"name"`
	Value int    `json:"value" yaml:"value"`
}

// --- NewWriter Factory Tests ---

func TestNewWriter_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	w, err := NewWriter(buf, FormatJSON)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if _, ok := w.(*JSONWriter); !ok {
		t.Errorf("expected *JSONWriter, got %T", w)
	}
}

func TestNewWriter_JSONL(t *testing.T) {
	buf := &bytes.Buffer{}
	w, err := NewWriter(buf, FormatJSONL)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if _, ok := w.(*JSONLWriter); !ok {
		t.Errorf("expected *JSONLWriter, got %T", w)
	}
}

func TestNewWriter_YAML(t *testing.T) {
	buf := &bytes.Buffer{}
	w, err := NewWriter(buf, FormatYAML)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if _, ok := w.(*YAMLWriter); !ok {
		t.Errorf("expected *YAMLWriter, got %T", w)
	}
}

func TestNewWriter_UnsupportedFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	_, err := NewWriter(buf, Format("unsupported"))
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}

	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected error containing 'unsupported', got %v", err)
	}
}

// --- JSONWriter Tests ---

func TestJSONWriter_Write_SingleItem(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, true, "  ")

	item := testItem{Name: "test", Value: 42}
	if err := w.Write(item); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Single item should be output directly, not as array
	var result testItem
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if result.Name != "test" || result.Value != 42 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestJSONWriter_Write_MultipleItems_OutputsArray(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, true, "  ")

	if err := w.Write(testItem{Name: "first", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Write(testItem{Name: "second", Value: 2}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Multiple items should be output as array
	var result []testItem
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	if result[0].Name != "first" || result[1].Name != "second" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestJSONWriter_WriteAll(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, false, "")

	items := []any{
		testItem{Name: "a", Value: 1},
		testItem{Name: "b", Value: 2},
		testItem{Name: "c", Value: 3},
	}

	if err := w.WriteAll(items); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	var result []testItem
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
}

func TestJSONWriter_Flush_PrettyPrint(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, true, "  ")

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	output := buf.String()

	// Pretty print should contain newlines and indentation
	if !strings.Contains(output, "\n") {
		t.Errorf("expected newlines in pretty output")
	}

	if !strings.Contains(output, "  ") {
		t.Errorf("expected indentation in pretty output")
	}
}

func TestJSONWriter_Flush_Compact(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, false, "")

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	output := buf.String()

	// Compact should not have pretty-print newlines (except the final one)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected single line in compact output, got %d lines", len(lines))
	}
}

func TestJSONWriter_Flush_CustomIndent(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, true, "\t")

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "\t") {
		t.Errorf("expected tab indentation, got %q", output)
	}
}

func TestJSONWriter_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, false, "")

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Close should flush
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Buffer should have content
	if buf.Len() == 0 {
		t.Error("expected output after Close()")
	}
}

// --- JSONLWriter Tests ---

func TestJSONLWriter_Write_SingleItem(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONLWriter(buf)

	if err := w.Write(testItem{Name: "test", Value: 42}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()

	// Should be a single JSON object followed by newline
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected newline at end of line")
	}

	var result testItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if result.Name != "test" || result.Value != 42 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestJSONLWriter_Write_MultipleItems_SeparateLines(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONLWriter(buf)

	if err := w.Write(testItem{Name: "first", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Write(testItem{Name: "second", Value: 2}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), output)
	}

	// Each line should be valid JSON
	for i, line := range lines {
		var item testItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Errorf("line %d is not valid JSON: %v", i, err)
		}
	}
}

func TestJSONLWriter_WriteAll(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONLWriter(buf)

	items := []any{
		testItem{Name: "a", Value: 1},
		testItem{Name: "b", Value: 2},
	}

	if err := w.WriteAll(items); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestJSONLWriter_Flush(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONLWriter(buf)

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Flush should ensure buffer is written (Write already flushes per-item)
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected output after Flush()")
	}
}

func TestJSONLWriter_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONLWriter(buf)

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected output after Close()")
	}
}

// --- YAMLWriter Tests ---

func TestYAMLWriter_Write_SingleItem(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewYAMLWriter(buf)

	item := testItem{Name: "test", Value: 42}
	if err := w.Write(item); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Should be valid YAML
	var result testItem
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if result.Name != "test" || result.Value != 42 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestYAMLWriter_Write_MultipleItems(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewYAMLWriter(buf)

	if err := w.Write(testItem{Name: "first", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Write(testItem{Name: "second", Value: 2}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Multiple items should be output as array
	var result []testItem
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
}

func TestYAMLWriter_WriteAll(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewYAMLWriter(buf)

	items := []any{
		testItem{Name: "a", Value: 1},
		testItem{Name: "b", Value: 2},
	}

	if err := w.WriteAll(items); err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	var result []testItem
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestYAMLWriter_Flush(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewYAMLWriter(buf)

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected output after Flush()")
	}

	// Check YAML uses 2-space indentation
	output := buf.String()
	if !strings.Contains(output, "name:") || !strings.Contains(output, "value:") {
		t.Errorf("expected YAML keys, got %q", output)
	}
}

func TestYAMLWriter_Close(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewYAMLWriter(buf)

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected output after Close()")
	}
}

// --- Option Tests ---

func TestWithPretty_Enabled(t *testing.T) {
	cfg := &writerConfig{}
	WithPretty(true)(cfg)

	if !cfg.pretty {
		t.Error("WithPretty(true) did not set pretty")
	}
}

func TestWithPretty_Disabled(t *testing.T) {
	cfg := &writerConfig{pretty: true}
	WithPretty(false)(cfg)

	if cfg.pretty {
		t.Error("WithPretty(false) did not unset pretty")
	}
}

func TestWithIndent_Custom(t *testing.T) {
	cfg := &writerConfig{}
	WithIndent("\t")(cfg)

	if cfg.indent != "\t" {
		t.Errorf("expected indent '\\t', got %q", cfg.indent)
	}
}

// --- Integration: NewWriter with Options ---

func TestNewWriter_WithOptions(t *testing.T) {
	buf := &bytes.Buffer{}

	w, err := NewWriter(buf, FormatJSON, WithPretty(false), WithIndent(""))
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if err := w.Write(testItem{Name: "test", Value: 1}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// With pretty=false, should be compact
	output := strings.TrimSpace(buf.String())
	if strings.Contains(output, "\n") {
		t.Errorf("expected compact output, got %q", output)
	}
}

// --- Edge Cases ---

func TestJSONWriter_Empty(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONWriter(buf, false, "")

	// Flush with no items
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Should output empty array or null
	output := strings.TrimSpace(buf.String())
	if output != "[]" && output != "null" {
		// Empty items list will output []
		var result []any
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Errorf("failed to unmarshal empty output: %v", err)
		}
	}
}

func TestJSONLWriter_Empty(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewJSONLWriter(buf)

	// Flush with no items
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// Should be empty
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestYAMLWriter_Empty(t *testing.T) {
	buf := &bytes.Buffer{}
	w := NewYAMLWriter(buf)

	// Flush with no items
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	// YAML encoder outputs [] for empty slice
	output := strings.TrimSpace(buf.String())
	if output != "[]" {
		var result []any
		if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Errorf("failed to unmarshal empty output: %v", err)
		}
	}
}
