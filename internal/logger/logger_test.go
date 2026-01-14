package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// resetLogger resets the logger to default state for test isolation
func resetLogger() {
	Init(Options{})
}

// --- Init Tests ---

func TestInit_DefaultLevel_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	// Info should be logged
	Info("test info")
	if !strings.Contains(buf.String(), "test info") {
		t.Error("Info message should be logged at default level")
	}

	buf.Reset()

	// Debug should NOT be logged at default level
	Debug("test debug")
	if strings.Contains(buf.String(), "test debug") {
		t.Error("Debug message should not be logged at default level")
	}
}

func TestInit_DebugLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Debug: true, Output: buf})
	defer resetLogger()

	// Debug should be logged when Debug=true
	Debug("test debug message")
	if !strings.Contains(buf.String(), "test debug message") {
		t.Error("Debug message should be logged when Debug=true")
	}
}

func TestInit_QuietLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Quiet: true, Output: buf})
	defer resetLogger()

	// Info should NOT be logged when Quiet=true
	Info("test info")
	if strings.Contains(buf.String(), "test info") {
		t.Error("Info message should not be logged when Quiet=true")
	}

	// Warn should NOT be logged when Quiet=true
	Warn("test warn")
	if strings.Contains(buf.String(), "test warn") {
		t.Error("Warn message should not be logged when Quiet=true")
	}

	// Error should be logged when Quiet=true
	Error("test error")
	if !strings.Contains(buf.String(), "test error") {
		t.Error("Error message should be logged when Quiet=true")
	}
}

func TestInit_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{JSON: true, Output: buf})
	defer resetLogger()

	Info("test message")

	output := buf.String()

	// JSON output should contain JSON structure
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Error("JSON format should produce JSON output")
	}

	// Should contain the message
	if !strings.Contains(output, "test message") {
		t.Error("JSON output should contain the message")
	}

	// Should contain level indicator
	if !strings.Contains(output, "level") {
		t.Error("JSON output should contain level field")
	}
}

func TestInit_TextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{JSON: false, Output: buf})
	defer resetLogger()

	Info("test message")

	output := buf.String()

	// Text output should contain the message
	if !strings.Contains(output, "test message") {
		t.Error("Text output should contain the message")
	}

	// Text output should contain level (INFO)
	if !strings.Contains(strings.ToUpper(output), "INFO") {
		t.Error("Text output should contain level INFO")
	}
}

func TestInit_CustomOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	Info("custom output test")

	if buf.Len() == 0 {
		t.Error("expected output to custom writer")
	}

	if !strings.Contains(buf.String(), "custom output test") {
		t.Error("expected message in custom output")
	}
}

// --- Log Function Tests ---

func TestDebug_NotLogged_AtInfoLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	Debug("should not appear")

	if strings.Contains(buf.String(), "should not appear") {
		t.Error("Debug should not be logged at Info level")
	}
}

func TestDebug_Logged_AtDebugLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Debug: true, Output: buf})
	defer resetLogger()

	Debug("should appear")

	if !strings.Contains(buf.String(), "should appear") {
		t.Error("Debug should be logged at Debug level")
	}
}

func TestInfo_LoggedAtInfoLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	Info("info message")

	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info should be logged at Info level")
	}
}

func TestWarn_LoggedAtInfoLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	Warn("warning message")

	if !strings.Contains(buf.String(), "warning message") {
		t.Error("Warn should be logged at Info level")
	}
}

func TestError_LoggedAtQuietLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Quiet: true, Output: buf})
	defer resetLogger()

	Error("error message")

	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error should be logged even at Quiet level")
	}
}

func TestError_LoggedAtInfoLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	Error("error message")

	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error should be logged at Info level")
	}
}

// --- With Tests ---

func TestWith_ReturnsLoggerWithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	logger := With("key", "value")
	if logger == nil {
		t.Fatal("With() returned nil")
	}

	logger.Info("test with attrs")

	output := buf.String()
	if !strings.Contains(output, "test with attrs") {
		t.Error("expected message in output")
	}

	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Error("expected attributes in output")
	}
}

// --- Context Tests ---

func TestDebugContext(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Debug: true, Output: buf})
	defer resetLogger()

	ctx := context.Background()
	DebugContext(ctx, "debug with context")

	if !strings.Contains(buf.String(), "debug with context") {
		t.Error("DebugContext should log message")
	}
}

func TestInfoContext(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	ctx := context.Background()
	InfoContext(ctx, "info with context")

	if !strings.Contains(buf.String(), "info with context") {
		t.Error("InfoContext should log message")
	}
}

func TestErrorContext(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	ctx := context.Background()
	ErrorContext(ctx, "error with context")

	if !strings.Contains(buf.String(), "error with context") {
		t.Error("ErrorContext should log message")
	}
}

// --- Structured Arguments Tests ---

func TestInfo_WithStructuredArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	Init(Options{Output: buf})
	defer resetLogger()

	Info("structured log", "count", 42, "name", "test")

	output := buf.String()
	if !strings.Contains(output, "structured log") {
		t.Error("expected message in output")
	}

	if !strings.Contains(output, "count") {
		t.Error("expected 'count' key in output")
	}

	if !strings.Contains(output, "42") {
		t.Error("expected '42' value in output")
	}

	if !strings.Contains(output, "name") {
		t.Error("expected 'name' key in output")
	}

	if !strings.Contains(output, "test") {
		t.Error("expected 'test' value in output")
	}
}

// --- Level Priority Tests ---

func TestQuiet_OverridesDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	// Both Debug and Quiet are set - Quiet should take precedence
	Init(Options{Debug: true, Quiet: true, Output: buf})
	defer resetLogger()

	Debug("debug message")
	Info("info message")
	Error("error message")

	output := buf.String()

	// Only Error should be logged
	if strings.Contains(output, "debug message") {
		t.Error("Debug should not be logged when Quiet=true")
	}

	if strings.Contains(output, "info message") {
		t.Error("Info should not be logged when Quiet=true")
	}

	if !strings.Contains(output, "error message") {
		t.Error("Error should be logged when Quiet=true")
	}
}
