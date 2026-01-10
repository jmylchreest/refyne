package output

import (
	"bufio"
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLWriter writes YAML output.
type YAMLWriter struct {
	w     *bufio.Writer
	items []any
}

// NewYAMLWriter creates a YAML writer.
func NewYAMLWriter(w io.Writer) *YAMLWriter {
	return &YAMLWriter{
		w:     bufio.NewWriter(w),
		items: make([]any, 0),
	}
}

// Write buffers a single item.
func (w *YAMLWriter) Write(data any) error {
	w.items = append(w.items, data)
	return nil
}

// WriteAll buffers multiple items.
func (w *YAMLWriter) WriteAll(data []any) error {
	w.items = append(w.items, data...)
	return nil
}

// Flush writes the buffered items as YAML.
func (w *YAMLWriter) Flush() error {
	encoder := yaml.NewEncoder(w.w)
	encoder.SetIndent(2)

	// If only one item, output it directly
	var err error
	if len(w.items) == 1 {
		err = encoder.Encode(w.items[0])
	} else {
		err = encoder.Encode(w.items)
	}

	if err != nil {
		return err
	}

	if err := encoder.Close(); err != nil {
		return err
	}

	return w.w.Flush()
}

// Close flushes and closes the writer.
func (w *YAMLWriter) Close() error {
	return w.Flush()
}
