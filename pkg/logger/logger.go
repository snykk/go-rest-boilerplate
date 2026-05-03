package logger

import (
	"context"
	"errors"
)

var (
	// ErrInvalidLoggerInstance is returned when an invalid logger instance type is provided
	ErrInvalidLoggerInstance = errors.New("invalid logger instance")

	// defaultLogger is the global logger instance
	defaultLogger Logger
)

// Log level constants
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelFatal = "fatal"
	LevelPanic = "panic"
)

// Logger instance types
const (
	InstanceZap int = iota
	// Another logger imlpementations in the future
	// ...
	// InstanceZerolog
	// InstanceLogrus
)

// Context keys for correlation IDs. TraceIDKey holds the W3C trace
// ID (auto-populated by OTel for cross-service link); RequestIDKey
// holds the external X-Request-ID echoed back to the client. Every
// *WithContext logging method emits both fields when present.
const (
	TraceIDKey   = "traceId"
	RequestIDKey = "request_id"
)

// Fields represents structured logging fields
type Fields map[string]interface{}

// Logger defines the interface for all logger implementations
type Logger interface {
	// Basic logging methods
	Debug(msg string, fields ...Fields)
	Info(msg string, fields ...Fields)
	Warn(msg string, fields ...Fields)
	Error(msg string, fields ...Fields)
	Fatal(msg string, fields ...Fields)
	Panic(msg string, fields ...Fields)

	// Formatted logging methods
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	// Context-aware logging methods
	DebugWithContext(ctx context.Context, msg string, fields ...Fields)
	InfoWithContext(ctx context.Context, msg string, fields ...Fields)
	WarnWithContext(ctx context.Context, msg string, fields ...Fields)
	ErrorWithContext(ctx context.Context, msg string, fields ...Fields)
	FatalWithContext(ctx context.Context, msg string, fields ...Fields)
	PanicWithContext(ctx context.Context, msg string, fields ...Fields)

	// Context-aware formatted logging methods
	DebugfWithContext(ctx context.Context, format string, args ...interface{})
	InfofWithContext(ctx context.Context, format string, args ...interface{})
	WarnfWithContext(ctx context.Context, format string, args ...interface{})
	ErrorfWithContext(ctx context.Context, format string, args ...interface{})
	FatalfWithContext(ctx context.Context, format string, args ...interface{})
	PanicfContext(ctx context.Context, format string, args ...interface{})

	// Chaining methods
	WithFields(fields Fields) Logger
	WithContext(ctx context.Context) Logger
}

// Config represents the logger configuration
type Config struct {
	Level             string
	EnableConsole     bool
	ConsoleJSONFormat bool
	EnableFile        bool
	FileJSONFormat    bool
	FileLocation      string
	AppName           string
	SamplingEnabled   bool
}

// NewLogger creates a new logger instance based on the provided configuration and type
func NewLogger(config Config, instanceType int) (Logger, error) {
	switch instanceType {
	case InstanceZap:
		return newZapLogger(config)
	default:
		return nil, ErrInvalidLoggerInstance
	}
}

// SetDefault sets the default global logger instance
func SetDefault(logger Logger) {
	defaultLogger = logger
}

// GetDefault returns the default global logger instance
func GetDefault() Logger {
	return defaultLogger
}

// InitDefault initializes the default global logger
func InitDefault(config Config, instanceType int) error {
	logger, err := NewLogger(config, instanceType)
	if err != nil {
		return err
	}
	SetDefault(logger)
	return nil
}

// Global convenience functions using the default logger

// Debug logs a debug message with optional fields
func Debug(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, fields...)
	}
}

// Info logs an info message with optional fields
func Info(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, fields...)
	}
}

// Warn logs a warning message with optional fields
func Warn(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, fields...)
	}
}

// Error logs an error message with optional fields
func Error(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, fields...)
	}
}

// Fatal logs a fatal message with optional fields and exits
func Fatal(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Fatal(msg, fields...)
	}
}

// Panic logs a panic message with optional fields and panics
func Panic(msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.Panic(msg, fields...)
	}
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debugf(format, args...)
	}
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Infof(format, args...)
	}
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warnf(format, args...)
	}
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Errorf(format, args...)
	}
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatalf(format, args...)
	}
}

// Panicf logs a formatted panic message and panics
func Panicf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Panicf(format, args...)
	}
}

// DebugWithContext logs a debug message with context and optional fields
func DebugWithContext(ctx context.Context, msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.DebugWithContext(ctx, msg, fields...)
	}
}

// InfoWithContext logs an info message with context and optional fields
func InfoWithContext(ctx context.Context, msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.InfoWithContext(ctx, msg, fields...)
	}
}

// WarnWithContext logs a warning message with context and optional fields
func WarnWithContext(ctx context.Context, msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.WarnWithContext(ctx, msg, fields...)
	}
}

// ErrorWithContext logs an error message with context and optional fields
func ErrorWithContext(ctx context.Context, msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.ErrorWithContext(ctx, msg, fields...)
	}
}

// FatalWithContext logs a fatal message with context and optional fields and exits
func FatalWithContext(ctx context.Context, msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.FatalWithContext(ctx, msg, fields...)
	}
}

// PanicWithContext logs a panic message with context and optional fields and panics
func PanicWithContext(ctx context.Context, msg string, fields ...Fields) {
	if defaultLogger != nil {
		defaultLogger.PanicWithContext(ctx, msg, fields...)
	}
}

// DebugfWithContext logs a formatted debug message with context
func DebugfWithContext(ctx context.Context, format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.DebugfWithContext(ctx, format, args...)
	}
}

// InfofWithContext logs a formatted info message with context
func InfofWithContext(ctx context.Context, format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.InfofWithContext(ctx, format, args...)
	}
}

// WarnfWithContext logs a formatted warning message with context
func WarnfWithContext(ctx context.Context, format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.WarnfWithContext(ctx, format, args...)
	}
}

// ErrorfWithContext logs a formatted error message with context
func ErrorfWithContext(ctx context.Context, format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.ErrorfWithContext(ctx, format, args...)
	}
}

// FatalfWithContext logs a formatted fatal message with context and exits
func FatalfWithContext(ctx context.Context, format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.FatalfWithContext(ctx, format, args...)
	}
}

// PanicfContext logs a formatted panic message with context and panics
func PanicfContext(ctx context.Context, format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.PanicfContext(ctx, format, args...)
	}
}

// WithFields returns a logger with the given fields attached
func WithFields(fields Fields) Logger {
	if defaultLogger != nil {
		return defaultLogger.WithFields(fields)
	}
	return nil
}

// WithContext returns a logger with context (including trace ID if present)
func WithContext(ctx context.Context) Logger {
	if defaultLogger != nil {
		return defaultLogger.WithContext(ctx)
	}
	return nil
}
