// Package output handles output formatting and writing.
package output

import (
	"fmt"
	"io"
)

// Format represents output format types.
type Format string

const (
	FormatJSON  Format = "json"
	FormatJSONL Format = "jsonl"
	FormatYAML  Format = "yaml"
)

// Writer handles output serialization.
type Writer interface {
	// Write outputs a single result.
	Write(data any) error

	// WriteAll outputs multiple results.
	WriteAll(data []any) error

	// Flush ensures all data is written.
	Flush() error

	// Close releases resources.
	Close() error
}

// WriterOption configures a writer.
type WriterOption func(*writerConfig)

type writerConfig struct {
	pretty bool
	indent string
}

// WithPretty enables pretty-printing.
func WithPretty(enabled bool) WriterOption {
	return func(c *writerConfig) {
		c.pretty = enabled
	}
}

// WithIndent sets the indentation string.
func WithIndent(indent string) WriterOption {
	return func(c *writerConfig) {
		c.indent = indent
	}
}

// NewWriter creates a writer for the specified format.
func NewWriter(w io.Writer, format Format, opts ...WriterOption) (Writer, error) {
	cfg := &writerConfig{
		pretty: true,
		indent: "  ",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	switch format {
	case FormatJSON:
		return NewJSONWriter(w, cfg.pretty, cfg.indent), nil
	case FormatJSONL:
		return NewJSONLWriter(w), nil
	case FormatYAML:
		return NewYAMLWriter(w), nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}
