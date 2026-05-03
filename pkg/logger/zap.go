package logger

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapLogger implements the Logger interface using zap as the underlying logging library.
// It maintains separate logger instances with optimized caller skip values to ensure
// accurate caller information in log entries across different usage patterns.
type zapLogger struct {
	base          *zap.Logger // Logger for direct method calls
	contextLogger *zap.Logger // Logger for context-aware method calls
	chainedLogger *zap.Logger // Logger for chained method calls
	fields        []zap.Field // Accumulated structured fields
	isChained     bool        // Indicates if logger was returned from WithContext/WithFields
}

// newZapLogger creates and configures a new zap logger instance.
// It initializes console and/or file outputs with JSON or console encoders,
// and sets up multiple logger instances with appropriate caller skip values
// to ensure accurate caller information in log entries.
func newZapLogger(config Config) (Logger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "function",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if config.ConsoleJSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	level := parseZapLevel(config.Level)
	cores := []zapcore.Core{}

	if config.EnableConsole {
		writer := zapcore.Lock(os.Stdout)
		cores = append(cores, zapcore.NewCore(encoder, writer, level))
	}

	if config.EnableFile && config.FileLocation != "" {
		file, err := os.OpenFile(config.FileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		var fileEncoder zapcore.Encoder
		if config.FileJSONFormat {
			fileEncoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			fileEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		writer := zapcore.AddSync(file)
		cores = append(cores, zapcore.NewCore(fileEncoder, writer, level))
	}

	if len(cores) == 0 {
		return nil, fmt.Errorf("no output configured (console or file must be enabled)")
	}

	core := zapcore.NewTee(cores...)

	baseLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(2),
	)

	contextLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(2),
	)

	chainedLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	if config.AppName != "" {
		baseLogger = baseLogger.With(zap.String("service", config.AppName))
		contextLogger = contextLogger.With(zap.String("service", config.AppName))
		chainedLogger = chainedLogger.With(zap.String("service", config.AppName))
	}

	return &zapLogger{
		base:          baseLogger,
		contextLogger: contextLogger,
		chainedLogger: chainedLogger,
		fields:        []zap.Field{},
		isChained:     false,
	}, nil
}

// parseZapLevel converts a string log level to zapcore.Level.
// Returns InfoLevel as default for unrecognized levels.
func parseZapLevel(level string) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	case LevelFatal:
		return zapcore.FatalLevel
	case LevelPanic:
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

// mergeFields combines accumulated fields with additional fields from method calls.
func (l *zapLogger) mergeFields(additional ...Fields) []zap.Field {
	totalSize := len(l.fields)
	for _, fieldMap := range additional {
		totalSize += len(fieldMap)
	}

	fields := make([]zap.Field, 0, totalSize)
	fields = append(fields, l.fields...)

	for _, fieldMap := range additional {
		for k, v := range fieldMap {
			fields = append(fields, zap.Any(k, v))
		}
	}

	return fields
}

// appendCorrelation pulls correlation IDs out of ctx and appends them
// as zap fields. Called by every *WithContext method so log entries
// share traceId (cross-service link) + request_id (client-facing).
func appendCorrelation(ctx context.Context, fields []zap.Field) []zap.Field {
	if traceID := GetTraceIDFromContext(ctx); traceID != "" && traceID != "unknown" {
		fields = append(fields, zap.String("traceId", traceID))
	}
	if requestID := GetRequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	return fields
}

// Basic logging methods

// Debug logs a debug-level message with optional structured fields.
func (l *zapLogger) Debug(msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	if l.isChained {
		l.chainedLogger.Debug(msg, allFields...)
	} else {
		l.base.Debug(msg, allFields...)
	}
}

// Info logs an info-level message with optional structured fields.
func (l *zapLogger) Info(msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	if l.isChained {
		l.chainedLogger.Info(msg, allFields...)
	} else {
		l.base.Info(msg, allFields...)
	}
}

// Warn logs a warning-level message with optional structured fields.
func (l *zapLogger) Warn(msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	if l.isChained {
		l.chainedLogger.Warn(msg, allFields...)
	} else {
		l.base.Warn(msg, allFields...)
	}
}

// Error logs an error-level message with optional structured fields.
func (l *zapLogger) Error(msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	if l.isChained {
		l.chainedLogger.Error(msg, allFields...)
	} else {
		l.base.Error(msg, allFields...)
	}
}

// Fatal logs a fatal-level message with optional structured fields and exits the program.
func (l *zapLogger) Fatal(msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	if l.isChained {
		l.chainedLogger.Fatal(msg, allFields...)
	} else {
		l.base.Fatal(msg, allFields...)
	}
}

// Panic logs a panic-level message with optional structured fields and panics.
func (l *zapLogger) Panic(msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	if l.isChained {
		l.chainedLogger.Panic(msg, allFields...)
	} else {
		l.base.Panic(msg, allFields...)
	}
}

// Formatted logging methods

// Debugf logs a formatted debug-level message using the accumulated fields.
func (l *zapLogger) Debugf(format string, args ...interface{}) {
	if l.isChained {
		l.chainedLogger.Debug(fmt.Sprintf(format, args...), l.fields...)
	} else {
		l.base.Debug(fmt.Sprintf(format, args...), l.fields...)
	}
}

// Infof logs a formatted info-level message using the accumulated fields.
func (l *zapLogger) Infof(format string, args ...interface{}) {
	if l.isChained {
		l.chainedLogger.Info(fmt.Sprintf(format, args...), l.fields...)
	} else {
		l.base.Info(fmt.Sprintf(format, args...), l.fields...)
	}
}

// Warnf logs a formatted warning-level message using the accumulated fields.
func (l *zapLogger) Warnf(format string, args ...interface{}) {
	if l.isChained {
		l.chainedLogger.Warn(fmt.Sprintf(format, args...), l.fields...)
	} else {
		l.base.Warn(fmt.Sprintf(format, args...), l.fields...)
	}
}

// Errorf logs a formatted error-level message using the accumulated fields.
func (l *zapLogger) Errorf(format string, args ...interface{}) {
	if l.isChained {
		l.chainedLogger.Error(fmt.Sprintf(format, args...), l.fields...)
	} else {
		l.base.Error(fmt.Sprintf(format, args...), l.fields...)
	}
}

// Fatalf logs a formatted fatal-level message using the accumulated fields and exits the program.
func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	if l.isChained {
		l.chainedLogger.Fatal(fmt.Sprintf(format, args...), l.fields...)
	} else {
		l.base.Fatal(fmt.Sprintf(format, args...), l.fields...)
	}
}

// Panicf logs a formatted panic-level message using the accumulated fields and panics.
func (l *zapLogger) Panicf(format string, args ...interface{}) {
	if l.isChained {
		l.chainedLogger.Panic(fmt.Sprintf(format, args...), l.fields...)
	} else {
		l.base.Panic(fmt.Sprintf(format, args...), l.fields...)
	}
}

// Context-aware logging methods

// DebugWithContext logs a debug-level message with context and optional structured fields.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) DebugWithContext(ctx context.Context, msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	allFields = appendCorrelation(ctx, allFields)
	l.contextLogger.Debug(msg, allFields...)
}

// InfoWithContext logs an info-level message with context and optional structured fields.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) InfoWithContext(ctx context.Context, msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	allFields = appendCorrelation(ctx, allFields)
	l.contextLogger.Info(msg, allFields...)
}

// WarnWithContext logs a warning-level message with context and optional structured fields.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) WarnWithContext(ctx context.Context, msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	allFields = appendCorrelation(ctx, allFields)
	l.contextLogger.Warn(msg, allFields...)
}

// ErrorWithContext logs an error-level message with context and optional structured fields.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) ErrorWithContext(ctx context.Context, msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	allFields = appendCorrelation(ctx, allFields)
	l.contextLogger.Error(msg, allFields...)
}

// FatalWithContext logs a fatal-level message with context and optional structured fields, then exits.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) FatalWithContext(ctx context.Context, msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	allFields = appendCorrelation(ctx, allFields)
	l.contextLogger.Fatal(msg, allFields...)
}

// PanicWithContext logs a panic-level message with context and optional structured fields, then panics.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) PanicWithContext(ctx context.Context, msg string, fields ...Fields) {
	allFields := l.mergeFields(fields...)
	allFields = appendCorrelation(ctx, allFields)
	l.contextLogger.Panic(msg, allFields...)
}

// Context-aware formatted logging methods

// DebugfWithContext logs a formatted debug-level message with context.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) DebugfWithContext(ctx context.Context, format string, args ...interface{}) {
	fields := make([]zap.Field, len(l.fields))
	copy(fields, l.fields)
	fields = appendCorrelation(ctx, fields)

	l.contextLogger.Debug(fmt.Sprintf(format, args...), fields...)
}

// InfofWithContext logs a formatted info-level message with context.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) InfofWithContext(ctx context.Context, format string, args ...interface{}) {
	fields := make([]zap.Field, len(l.fields))
	copy(fields, l.fields)
	fields = appendCorrelation(ctx, fields)

	l.contextLogger.Info(fmt.Sprintf(format, args...), fields...)
}

// WarnfWithContext logs a formatted warning-level message with context.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) WarnfWithContext(ctx context.Context, format string, args ...interface{}) {
	fields := make([]zap.Field, len(l.fields))
	copy(fields, l.fields)
	fields = appendCorrelation(ctx, fields)

	l.contextLogger.Warn(fmt.Sprintf(format, args...), fields...)
}

// ErrorfWithContext logs a formatted error-level message with context.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) ErrorfWithContext(ctx context.Context, format string, args ...interface{}) {
	fields := make([]zap.Field, len(l.fields))
	copy(fields, l.fields)
	fields = appendCorrelation(ctx, fields)

	l.contextLogger.Error(fmt.Sprintf(format, args...), fields...)
}

// FatalfWithContext logs a formatted fatal-level message with context, then exits.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) FatalfWithContext(ctx context.Context, format string, args ...interface{}) {
	fields := make([]zap.Field, len(l.fields))
	copy(fields, l.fields)
	fields = appendCorrelation(ctx, fields)

	l.contextLogger.Fatal(fmt.Sprintf(format, args...), fields...)
}

// PanicfContext logs a formatted panic-level message with context, then panics.
// Automatically extracts and includes traceId from context if available.
func (l *zapLogger) PanicfContext(ctx context.Context, format string, args ...interface{}) {
	fields := make([]zap.Field, len(l.fields))
	copy(fields, l.fields)
	fields = appendCorrelation(ctx, fields)

	l.contextLogger.Panic(fmt.Sprintf(format, args...), fields...)
}

// Chaining methods

// WithFields returns a new logger instance with the provided fields merged
// into the existing accumulated fields. Supports method chaining.
func (l *zapLogger) WithFields(fields Fields) Logger {
	newFields := make([]zap.Field, 0, len(l.fields)+len(fields))
	newFields = append(newFields, l.fields...)

	for k, v := range fields {
		newFields = append(newFields, zap.Any(k, v))
	}

	return &zapLogger{
		base:          l.base,
		contextLogger: l.contextLogger,
		chainedLogger: l.chainedLogger,
		fields:        newFields,
		isChained:     true,
	}
}

// WithContext returns a new logger instance with both correlation IDs
// (traceId from OTel and request_id from X-Request-ID) baked in, when
// present in ctx. Supports method chaining.
func (l *zapLogger) WithContext(ctx context.Context) Logger {
	newFields := make([]zap.Field, 0, len(l.fields)+2)
	newFields = append(newFields, l.fields...)
	newFields = appendCorrelation(ctx, newFields)

	return &zapLogger{
		base:          l.base,
		contextLogger: l.contextLogger,
		chainedLogger: l.chainedLogger,
		fields:        newFields,
		isChained:     true,
	}
}
