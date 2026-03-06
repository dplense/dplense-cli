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
	Print(format string, args ...interface{}) // User-facing output without timestamp/level
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
	level      Level     // Terminal level threshold (default: Warn — only warnings/errors in terminal)
	fileLevel  Level     // File level threshold (default: Info — full audit trail in file)
	output     io.Writer // Terminal writer (os.Stderr or io.Discard for quiet mode)
	fileOutput io.Writer // File writer for audit trail (nil if file logging disabled)
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

// NewWithFileWriter creates a logger that writes to both a terminal and a file.
// termLevel controls what appears in the terminal (default: Warn — keeps CLI clean).
// fileLevel controls what goes into the audit log file (default: Info — detailed trail).
// The file receives plain text without ANSI color codes. Pass nil for fileWriter to disable file logging.
func NewWithFileWriter(termLevel Level, fileLevel Level, jsonOutput bool, termWriter io.Writer, fileWriter io.Writer) Logger {
	return &loggerImpl{
		level:      termLevel,
		fileLevel:  fileLevel,
		output:     termWriter,
		fileOutput: fileWriter,
		jsonOutput: jsonOutput,
		fields:     make(map[string]interface{}),
	}
}

// WithContext returns a logger with context fields
func (l *loggerImpl) WithContext(ctx context.Context) Logger {
	newLogger := &loggerImpl{
		level:      l.level,
		fileLevel:  l.fileLevel,
		output:     l.output,
		fileOutput: l.fileOutput,
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

// log writes a log entry with independent level thresholds for terminal and file.
// Terminal only shows messages at l.level or above (default: Warn — keeps CLI clean).
// File captures messages at l.fileLevel or above (default: Info — full audit trail).
func (l *loggerImpl) log(level Level, format string, args ...interface{}) {
	writeTerminal := level >= l.level
	writeFile := l.fileOutput != nil && level >= l.fileLevel

	if !writeTerminal && !writeFile {
		return
	}

	message := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Fields:    l.fields,
	}

	// Terminal (only if level passes terminal threshold)
	if writeTerminal {
		if l.jsonOutput {
			l.writeJSON(entry)
		} else {
			l.writeText(entry)
		}
	}

	// File (plain text, no ANSI — only if level passes file threshold)
	if writeFile {
		l.writeFile(entry)
	}
}

// writeJSON writes a log entry in JSON format to the terminal
func (l *loggerImpl) writeJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to text if JSON marshaling fails
		l.writeText(entry)
		return
	}
	fmt.Fprintln(l.output, string(data))
}

// writeText writes a log entry in human-readable format to the terminal (with ANSI colors)
func (l *loggerImpl) writeText(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

	fieldsStr := ""
	if len(entry.Fields) > 0 {
		fieldsStr = fmt.Sprintf(" %+v", entry.Fields)
	}

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
	fmt.Fprintf(l.output, "[%s] %s: %s%s\n", timestamp, levelStr, entry.Message, fieldsStr)
}

// writeFile writes a log entry in plain text to the file (no ANSI codes, all levels)
func (l *loggerImpl) writeFile(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

	fieldsStr := ""
	if len(entry.Fields) > 0 {
		fieldsStr = fmt.Sprintf(" %+v", entry.Fields)
	}

	fmt.Fprintf(l.fileOutput, "[%s] %s: %s%s\n", timestamp, entry.Level, entry.Message, fieldsStr)
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

// Print writes a user-facing message to the terminal without timestamp or level prefix.
// Respects --quiet mode (output goes to io.Discard).
// Does NOT write to the log file — use Info() for audit trail entries.
func (l *loggerImpl) Print(format string, args ...interface{}) {
	fmt.Fprintf(l.output, format+"\n", args...)
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
