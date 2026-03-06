package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &loggerImpl{
		level:      LevelDebug,
		output:     buf,
		jsonOutput: false,
		fields:     make(map[string]interface{}),
	}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Error("Expected DEBUG in output")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("Expected INFO in output")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("Expected WARN in output")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("Expected ERROR in output")
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &loggerImpl{
		level:      LevelWarn,
		output:     buf,
		jsonOutput: false,
		fields:     make(map[string]interface{}),
	}

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	if strings.Contains(output, "DEBUG") {
		t.Error("DEBUG should be filtered out")
	}
	if strings.Contains(output, "INFO") {
		t.Error("INFO should be filtered out")
	}
	if !strings.Contains(output, "WARN") {
		t.Error("WARN should be included")
	}
	if !strings.Contains(output, "ERROR") {
		t.Error("ERROR should be included")
	}
}

func TestLogger_JSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &loggerImpl{
		level:      LevelInfo,
		output:     buf,
		jsonOutput: true,
		fields:     make(map[string]interface{}),
	}

	logger.Info("test message")

	output := buf.String()
	var entry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", entry.Message)
	}
}

func TestLogger_WithContext(t *testing.T) {
	logger := New(LevelInfo, false)
	ctxLogger := logger.WithContext(context.Background())

	if ctxLogger == nil {
		t.Error("WithContext should return a logger")
	}
}

func TestLogger_FileOutput(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	// Both at Debug level to verify all messages reach both outputs
	l := NewWithFileWriter(LevelDebug, LevelDebug, false, termBuf, fileBuf)

	l.Debug("debug msg")
	l.Info("info msg")
	l.Warn("warn msg")
	l.Error("error msg")

	// Both buffers should have all messages
	termOut := termBuf.String()
	fileOut := fileBuf.String()

	for _, level := range []string{"DEBUG", "INFO", "WARN", "ERROR"} {
		if !strings.Contains(termOut, level) {
			t.Errorf("Terminal output missing %s", level)
		}
		if !strings.Contains(fileOut, level) {
			t.Errorf("File output missing %s", level)
		}
	}

	// File output should NOT contain ANSI escape codes (ESC = \x1b)
	if strings.Contains(fileOut, "\x1b") {
		t.Error("File output should not contain ANSI escape codes")
	}

	// File output should have timestamps
	if !strings.Contains(fileOut, "] DEBUG:") {
		t.Error("File output should have plain level tags")
	}
}

func TestLogger_FileOutput_Nil(t *testing.T) {
	termBuf := &bytes.Buffer{}
	l := NewWithFileWriter(LevelInfo, LevelInfo, false, termBuf, nil)

	// Should not panic with nil file writer
	l.Info("test message")
	l.Warn("warn message")
	l.Print("print message")

	if !strings.Contains(termBuf.String(), "test message") {
		t.Error("Terminal output should still work with nil file writer")
	}
}

func TestLogger_Print_TerminalOnly(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	l := NewWithFileWriter(LevelWarn, LevelInfo, false, termBuf, fileBuf)

	l.Print("Found %d drives", 10)

	// Terminal: raw message without timestamp/level
	termOut := termBuf.String()
	if termOut != "Found 10 drives\n" {
		t.Errorf("Terminal Print output should be raw, got: %q", termOut)
	}

	// File: Print does NOT write to file
	fileOut := fileBuf.String()
	if fileOut != "" {
		t.Errorf("Print should not write to file, got: %q", fileOut)
	}
}

func TestLogger_WithContext_PropagatesFileOutput(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	l := NewWithFileWriter(LevelInfo, LevelInfo, false, termBuf, fileBuf)

	ctxLogger := l.WithContext(context.Background())

	// Log via context logger — should write to both buffers
	ctxLogger.Info("context message")

	if !strings.Contains(termBuf.String(), "context message") {
		t.Error("Terminal should receive messages from context logger")
	}
	if !strings.Contains(fileBuf.String(), "context message") {
		t.Error("File should receive messages from context logger")
	}
}

func TestLogger_FileOutput_JSON(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	l := NewWithFileWriter(LevelInfo, LevelInfo, true, termBuf, fileBuf)

	l.Info("json test")

	// Terminal should have JSON
	var termEntry LogEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(termBuf.String())), &termEntry); err != nil {
		t.Fatalf("Failed to unmarshal terminal JSON: %v", err)
	}
	if termEntry.Message != "json test" {
		t.Errorf("Terminal JSON should contain message, got: %s", termEntry.Message)
	}

	// File always gets plain text (audit trail), not JSON
	fileOut := fileBuf.String()
	if !strings.Contains(fileOut, "INFO: json test") {
		t.Errorf("File should contain plain text entry, got: %q", fileOut)
	}
}

func TestLogger_DualLevels_TermWarnFileInfo(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	// Terminal: Warn (only warnings/errors), File: Info (full audit trail)
	l := NewWithFileWriter(LevelWarn, LevelInfo, false, termBuf, fileBuf)

	l.Debug("debug secret")
	l.Info("info audit step")
	l.Warn("warn msg")
	l.Error("error msg")

	termOut := termBuf.String()
	fileOut := fileBuf.String()

	// Terminal should NOT have Debug or Info (clean CLI)
	if strings.Contains(termOut, "debug secret") {
		t.Error("Terminal should not show Debug when termLevel=Warn")
	}
	if strings.Contains(termOut, "info audit step") {
		t.Error("Terminal should not show Info when termLevel=Warn")
	}

	// Terminal SHOULD have Warn and Error
	if !strings.Contains(termOut, "WARN") {
		t.Error("Terminal should show Warn")
	}
	if !strings.Contains(termOut, "ERROR") {
		t.Error("Terminal should show Error")
	}

	// File should NOT have Debug (fileLevel=Info)
	if strings.Contains(fileOut, "debug secret") {
		t.Error("File should not show Debug when fileLevel=Info")
	}

	// File SHOULD have Info, Warn, Error (full audit trail)
	for _, expected := range []string{"INFO: info audit step", "WARN: warn msg", "ERROR: error msg"} {
		if !strings.Contains(fileOut, expected) {
			t.Errorf("File should contain %q", expected)
		}
	}
}

func TestLogger_DualLevels_DebugMode(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	// Debug mode: both terminal and file get everything
	l := NewWithFileWriter(LevelDebug, LevelDebug, false, termBuf, fileBuf)

	l.Debug("debug detail")
	l.Info("info step")

	termOut := termBuf.String()
	fileOut := fileBuf.String()

	// Both should have Debug and Info
	if !strings.Contains(termOut, "debug detail") {
		t.Error("Terminal should show Debug in debug mode")
	}
	if !strings.Contains(fileOut, "debug detail") {
		t.Error("File should show Debug in debug mode")
	}
	if !strings.Contains(termOut, "info step") {
		t.Error("Terminal should show Info in debug mode")
	}
	if !strings.Contains(fileOut, "info step") {
		t.Error("File should show Info in debug mode")
	}
}

func TestLogger_FileOnlyInfo_TerminalClean(t *testing.T) {
	termBuf := &bytes.Buffer{}
	fileBuf := &bytes.Buffer{}
	// Default production config: terminal=Warn, file=Info
	l := NewWithFileWriter(LevelWarn, LevelInfo, false, termBuf, fileBuf)

	l.Info("API call: Files.List returned 100 files")
	l.Info("Scanning target 1/5: Engineering Drive")
	l.Print("Found 5 shared drives to scan")

	termOut := termBuf.String()
	fileOut := fileBuf.String()

	// Terminal: only Print() output, NO Info messages
	if strings.Contains(termOut, "API call") {
		t.Error("Terminal should not show Info-level API details")
	}
	if strings.Contains(termOut, "Scanning target") {
		t.Error("Terminal should not show Info-level scan progress")
	}
	if !strings.Contains(termOut, "Found 5 shared drives to scan") {
		t.Error("Terminal should show Print() output")
	}

	// File: Info-level audit trail (Print does NOT go to file)
	if !strings.Contains(fileOut, "INFO: API call: Files.List returned 100 files") {
		t.Error("File should capture Info-level API details")
	}
	if !strings.Contains(fileOut, "INFO: Scanning target 1/5: Engineering Drive") {
		t.Error("File should capture Info-level scan progress")
	}
	if strings.Contains(fileOut, "Found 5 shared drives") {
		t.Error("Print() should not write to file")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"invalid", LevelInfo}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseLevel(tt.input); got != tt.expected {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
