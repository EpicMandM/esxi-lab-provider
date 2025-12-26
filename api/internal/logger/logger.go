package logger

import (
	"fmt"
	"io"
	"os"
)

// Logger provides structured logging for journald
type Logger struct {
	writer io.Writer
}

// New creates a new logger instance
func New() *Logger {
	return &Logger{
		writer: os.Stdout,
	}
}

// NewWithWriter creates a logger with a custom writer
func NewWithWriter(w io.Writer) *Logger {
	return &Logger{
		writer: w,
	}
}

// Info logs informational messages
func (l *Logger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields...)
}

// Error logs error messages
func (l *Logger) Error(msg string, fields ...Field) {
	l.log("ERROR", msg, fields...)
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log("WARNING", msg, fields...)
}

// Debug logs debug messages
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log("DEBUG", msg, fields...)
}

func (l *Logger) log(level, msg string, fields ...Field) {
	output := fmt.Sprintf("LEVEL=%s MESSAGE=%s", level, msg)
	for _, field := range fields {
		output += fmt.Sprintf(" %s=%v", field.Key, field.Value)
	}
	_, _ = fmt.Fprintln(l.writer, output)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new field (shorthand)
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Common field constructors
func Action(value string) Field     { return F("ACTION", value) }
func Status(value string) Field     { return F("STATUS", value) }
func VM(value string) Field         { return F("VM", value) }
func User(value string) Field       { return F("USER", value) }
func Count(value int) Field         { return F("COUNT", value) }
func Error(value error) Field       { return F("ERROR", value) }
func Snapshot(value string) Field   { return F("SNAPSHOT", value) }
func Password(value string) Field   { return F("PASSWORD", value) }
func VMIndex(value int) Field       { return F("VM_INDEX", value) }
func Events(value int) Field        { return F("EVENTS", value) }
func Restored(value int) Field      { return F("RESTORED", value) }
func Failed(value int) Field        { return F("FAILED", value) }
func TimeWindow(value string) Field { return F("TIME_WINDOW", value) }
func Reason(value string) Field     { return F("REASON", value) }
