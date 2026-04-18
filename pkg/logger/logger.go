package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// Level represents log level
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Config holds logger configuration
type Config struct {
	Level  string
	Format string
	Output io.Writer
}

// Logger represents a structured logger
type Logger struct {
	level  Level
	format string
	logger *log.Logger
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// New creates a new logger
func New(cfg Config) *Logger {
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	level := parseLevel(cfg.Level)

	return &Logger{
		level:  level,
		format: cfg.Format,
		logger: log.New(output, "", log.LstdFlags),
	}
}

// WithComponent creates a child logger with a component prefix
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		level:  l.level,
		format: l.format,
		logger: log.New(l.logger.Writer(), fmt.Sprintf("[%s] ", component), log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	if l.level <= DebugLevel {
		l.log("DEBUG", msg, fields...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	if l.level <= InfoLevel {
		l.log("INFO", msg, fields...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	if l.level <= WarnLevel {
		l.log("WARN", msg, fields...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Field) {
	if l.level <= ErrorLevel {
		l.log("ERROR", msg, fields...)
	}
}

func (l *Logger) log(level, msg string, fields ...Field) {
	if len(fields) == 0 {
		l.logger.Printf("[%s] %s", level, msg)
		return
	}

	var fieldStrs []string
	for _, f := range fields {
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", f.Key, f.Value))
	}

	l.logger.Printf("[%s] %s %s", level, msg, strings.Join(fieldStrs, " "))
}

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Field constructors

// String creates a string field
func String(key, val string) Field {
	return Field{Key: key, Value: val}
}

// Int creates an int field
func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

// Int64 creates an int64 field
func Int64(key string, val int64) Field {
	return Field{Key: key, Value: val}
}

// Uint64 creates a uint64 field
func Uint64(key string, val uint64) Field {
	return Field{Key: key, Value: val}
}

// Bool creates a bool field
func Bool(key string, val bool) Field {
	return Field{Key: key, Value: val}
}

// Uint creates a uint field
func Uint(key string, val uint) Field {
	return Field{Key: key, Value: val}
}

// Uint32 creates a uint32 field
func Uint32(key string, val uint32) Field {
	return Field{Key: key, Value: val}
}

// Float64 creates a float64 field
func Float64(key string, val float64) Field {
	return Field{Key: key, Value: val}
}

// Error creates an error field
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: "nil"}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value
func Any(key string, val interface{}) Field {
	return Field{Key: key, Value: val}
}
