package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Level represents the log level
type Level int

const (
	// LevelDebug is the debug log level
	LevelDebug Level = iota
	// LevelInfo is the info log level
	LevelInfo
	// LevelWarn is the warning log level
	LevelWarn
	// LevelError is the error log level
	LevelError
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger interface for structured logging
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	WithContext(ctx context.Context) Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// loggerImpl implements the Logger interface
type loggerImpl struct {
	level      Level
	output     io.Writer
	jsonOutput bool
	fields     map[string]interface{}
}

// New creates a new logger instance
func New(level Level, jsonOutput bool) Logger {
	return &loggerImpl{
		level:      level,
		output:     os.Stderr,
		jsonOutput: jsonOutput,
		fields:     make(map[string]interface{}),
	}
}

// NewWithWriter creates a new logger instance with a custom writer
func NewWithWriter(level Level, jsonOutput bool, writer io.Writer) Logger {
	return &loggerImpl{
		level:      level,
		output:     writer,
		jsonOutput: jsonOutput,
		fields:     make(map[string]interface{}),
	}
}

// WithContext returns a logger with context fields
func (l *loggerImpl) WithContext(ctx context.Context) Logger {
	newLogger := &loggerImpl{
		level:      l.level,
		output:     l.output,
		jsonOutput: l.jsonOutput,
		fields:     make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add context fields if available
	if ctx != nil {
		// Extract request ID or other context values if needed
		// This is a placeholder for future context extraction
	}

	return newLogger
}

// log writes a log entry
func (l *loggerImpl) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	message := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Fields:    l.fields,
	}

	if l.jsonOutput {
		l.writeJSON(entry)
	} else {
		l.writeText(entry)
	}
}

// writeJSON writes a log entry in JSON format
func (l *loggerImpl) writeJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to text if JSON marshaling fails
		l.writeText(entry)
		return
	}
	fmt.Fprintln(l.output, string(data))
}

// writeText writes a log entry in human-readable format
func (l *loggerImpl) writeText(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	
	// Apply color styling based on log level
	var levelStr string
	switch entry.Level {
	case "WARN":
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
		levelStr = warnStyle.Render(entry.Level)
	case "ERROR":
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
		levelStr = errorStyle.Render(entry.Level)
	default:
		levelStr = entry.Level // No color for DEBUG and INFO
	}
	
	fmt.Fprintf(l.output, "[%s] %s: %s", timestamp, levelStr, entry.Message)
	if len(entry.Fields) > 0 {
		fmt.Fprintf(l.output, " %+v", entry.Fields)
	}
	fmt.Fprintln(l.output)
}

// Debug logs a debug message
func (l *loggerImpl) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *loggerImpl) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *loggerImpl) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *loggerImpl) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// ParseLevel parses a log level string
func ParseLevel(level string) Level {
	switch level {
	case "debug", "DEBUG":
		return LevelDebug
	case "info", "INFO":
		return LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return LevelWarn
	case "error", "ERROR":
		return LevelError
	default:
		return LevelInfo // Default to info
	}
}
