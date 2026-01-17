// Package logger provides structured logging for refyne.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger *slog.Logger
	mu            sync.RWMutex
)

func init() {
	// Default to discarding debug logs
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Init initializes the logger with the specified options.
func Init(opts Options) {
	mu.Lock()
	defer mu.Unlock()

	// If a custom logger is provided, use it directly
	if opts.Logger != nil {
		defaultLogger = opts.Logger
		return
	}

	level := slog.LevelInfo
	if opts.Debug {
		level = slog.LevelDebug
	}
	if opts.Quiet {
		level = slog.LevelError
	}

	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if opts.JSON {
		handler = slog.NewJSONHandler(output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(output, handlerOpts)
	}

	defaultLogger = slog.New(handler)
}

// SetLogger sets a custom slog.Logger to be used by refyne.
// This allows integration with your application's existing logging system.
func SetLogger(l *slog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = l
}

// Options configures the logger.
type Options struct {
	Debug  bool         // Enable debug level logging
	Quiet  bool         // Only show errors
	JSON   bool         // Output as JSON
	Output io.Writer    // Output destination (default: stderr)
	Logger *slog.Logger // Custom logger (overrides all other options)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.Error(msg, args...)
}

// With returns a logger with the given attributes.
func With(args ...any) *slog.Logger {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	return l.With(args...)
}

// DebugContext logs a debug message with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.DebugContext(ctx, msg, args...)
}

// InfoContext logs an info message with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.InfoContext(ctx, msg, args...)
}

// ErrorContext logs an error message with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	l.ErrorContext(ctx, msg, args...)
}
