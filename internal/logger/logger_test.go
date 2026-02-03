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
