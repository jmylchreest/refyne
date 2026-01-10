package output

import (
	"bufio"
	"encoding/json"
	"io"
)

// JSONWriter writes JSON output.
type JSONWriter struct {
	w      *bufio.Writer
	pretty bool
	indent string
	items  []any
}

// NewJSONWriter creates a JSON writer.
func NewJSONWriter(w io.Writer, pretty bool, indent string) *JSONWriter {
	return &JSONWriter{
		w:      bufio.NewWriter(w),
		pretty: pretty,
		indent: indent,
		items:  make([]any, 0),
	}
}

// Write buffers a single item for JSON array output.
func (w *JSONWriter) Write(data any) error {
	w.items = append(w.items, data)
	return nil
}

// WriteAll writes all items at once.
func (w *JSONWriter) WriteAll(data []any) error {
	w.items = append(w.items, data...)
	return nil
}

// Flush writes the buffered items as a JSON array.
func (w *JSONWriter) Flush() error {
	var output []byte
	var err error

	// If only one item, output it directly (not as array)
	if len(w.items) == 1 {
		if w.pretty {
			output, err = json.MarshalIndent(w.items[0], "", w.indent)
		} else {
			output, err = json.Marshal(w.items[0])
		}
	} else {
		if w.pretty {
			output, err = json.MarshalIndent(w.items, "", w.indent)
		} else {
			output, err = json.Marshal(w.items)
		}
	}

	if err != nil {
		return err
	}

	if _, err := w.w.Write(output); err != nil {
		return err
	}
	if _, err := w.w.WriteString("\n"); err != nil {
		return err
	}

	return w.w.Flush()
}

// Close flushes and closes the writer.
func (w *JSONWriter) Close() error {
	return w.Flush()
}

// JSONLWriter writes newline-delimited JSON (JSONL).
type JSONLWriter struct {
	w *bufio.Writer
}

// NewJSONLWriter creates a JSONL writer.
func NewJSONLWriter(w io.Writer) *JSONLWriter {
	return &JSONLWriter{
		w: bufio.NewWriter(w),
	}
}

// Write writes a single item as a JSON line.
func (w *JSONLWriter) Write(data any) error {
	output, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := w.w.Write(output); err != nil {
		return err
	}
	if _, err := w.w.WriteString("\n"); err != nil {
		return err
	}

	return w.w.Flush()
}

// WriteAll writes multiple items as JSON lines.
func (w *JSONLWriter) WriteAll(data []any) error {
	for _, item := range data {
		if err := w.Write(item); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes the buffer.
func (w *JSONLWriter) Flush() error {
	return w.w.Flush()
}

// Close flushes the writer.
func (w *JSONLWriter) Close() error {
	return w.Flush()
}
