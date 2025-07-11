package logtrace

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"runtime"
)

type LogLevel func(msg string, keysAndValues ...interface{})

// ContextKey is the type used for storing values in context
type ContextKey string

// CorrelationIDKey is the key for storing correlation ID in context
const CorrelationIDKey ContextKey = "correlation_id"

// Setup initializes the logger with a specified log level
func Setup(serviceName, env string, level string) {
	var slogLevel slog.Level
	switch level {
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	case "debug":
		slogLevel = slog.LevelDebug
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     slogLevel,
	}

	hostname, _ := os.Hostname()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	logger = logger.With("hostname", hostname).With("service", fmt.Sprintf("%s-%s", serviceName, env))
	slog.SetDefault(logger)
}

// log logs a message with additional fields using the specified log function.
func log(logLevel slog.Level, logFunc LogLevel, message string, fields Fields) {
	fieldArgs := make([]interface{}, 0, len(fields)*2+1)

	// Only attach source info for TRACE/DEBUG level
	if logLevel == slog.LevelDebug || logLevel == slog.LevelError {
		if pc, file, line, ok := runtime.Caller(2); ok {
			details := runtime.FuncForPC(pc)
			fieldArgs = append(fieldArgs, slog.Group(
				"source",
				slog.String("filename", file),
				slog.Int("lineno", line),
				slog.String("function", details.Name()),
			))
		}
	}

	for key, value := range fields {
		fieldArgs = append(fieldArgs, slog.Any(key, value))
	}

	logFunc(message, fieldArgs...)
}

// extractCorrelationID retrieves the correlation ID from context
func extractCorrelationID(ctx context.Context) string {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return correlationID
	}
	return "unknown"
}

// addCorrelationID ensures logs include the correlation ID without modifying the input fields.
func addCorrelationID(ctx context.Context, fields Fields) Fields {
	newFields := make(Fields, len(fields)+1) // Copy fields to avoid mutation
	maps.Copy(newFields, fields)
	newFields[FieldCorrelationID] = extractCorrelationID(ctx)
	return newFields
}

// Error logs an error message with structured fields.
func Error(ctx context.Context, message string, fields Fields) {
	log(slog.LevelError, slog.Error, message, addCorrelationID(ctx, fields))
}

// Info logs an informational message with structured fields.
func Info(ctx context.Context, message string, fields Fields) {
	log(slog.LevelInfo, slog.Info, message, addCorrelationID(ctx, fields))
}

// Warn logs a warning message with structured fields.
func Warn(ctx context.Context, message string, fields Fields) {
	log(slog.LevelWarn, slog.Warn, message, addCorrelationID(ctx, fields))
}

// Debug logs a debug message with structured fields.
func Debug(ctx context.Context, message string, fields Fields) {
	log(slog.LevelDebug, slog.Debug, message, addCorrelationID(ctx, fields))
}

// CtxWithCorrelationID stores a correlation ID inside the context.
func CtxWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}
