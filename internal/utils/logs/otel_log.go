// Package logs provides OpenTelemetry-based observability with Zap structured logging
// following best practices for distributed tracing and high-performance logging with ELK stack integration
package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/configs"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	ServiceNameDB = "database"
)

// LogLevel represents log level enumeration
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// StructuredLog represents the structure for JSON logs compatible with ELK stack
type StructuredLog struct {
	Timestamp      time.Time              `json:"@timestamp"`
	Level          LogLevel               `json:"level"`
	Message        string                 `json:"message"`
	ServiceName    string                 `json:"service.name"`
	ServiceVersion string                 `json:"service.version"`
	Environment    string                 `json:"environment"`
	TraceID        string                 `json:"trace.id,omitempty"`
	SpanID         string                 `json:"span.id,omitempty"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
	Error          *ErrorInfo             `json:"error,omitempty"`
}

// ZapLogger wraps Zap logger with OpenTelemetry integration
type ZapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

// Info logs an info level message
func (z *ZapLogger) Info(msg string, fields ...zap.Field) {
	z.logger.Info(msg, fields...)
}

// Infof logs an info level message with formatting
func (z *ZapLogger) Infof(template string, args ...interface{}) {
	z.sugar.Infof(template, args...)
}

// Debug logs a debug level message
func (z *ZapLogger) Debug(msg string, fields ...zap.Field) {
	z.logger.Debug(msg, fields...)
}

// Debugf logs a debug level message with formatting
func (z *ZapLogger) Debugf(template string, args ...interface{}) {
	z.sugar.Debugf(template, args...)
}

// Warn logs a warn level message
func (z *ZapLogger) Warn(msg string, fields ...zap.Field) {
	z.logger.Warn(msg, fields...)
}

// Warnf logs a warn level message with formatting
func (z *ZapLogger) Warnf(template string, args ...interface{}) {
	z.sugar.Warnf(template, args...)
}

// Error logs an error level message
func (z *ZapLogger) Error(msg string, fields ...zap.Field) {
	z.logger.Error(msg, fields...)
}

// Errorf logs an error level message with formatting
func (z *ZapLogger) Errorf(template string, args ...interface{}) {
	z.sugar.Errorf(template, args...)
}

// Fatal logs a fatal level message and exits
func (z *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	z.logger.Fatal(msg, fields...)
}

// Fatalf logs a fatal level message with formatting and exits
func (z *ZapLogger) Fatalf(template string, args ...interface{}) {
	z.sugar.Fatalf(template, args...)
}

// With adds structured context to the logger
func (z *ZapLogger) With(fields ...zap.Field) *ZapLogger {
	return &ZapLogger{
		logger: z.logger.With(fields...),
		sugar:  z.logger.With(fields...).Sugar(),
	}
}

// WithContext adds context fields like trace ID and span ID from OpenTelemetry context
func (z *ZapLogger) WithContext(ctx context.Context) *ZapLogger {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return z.With(
			zap.String("trace.id", span.SpanContext().TraceID().String()),
			zap.String("span.id", span.SpanContext().SpanID().String()),
		)
	}
	return z
}

// ErrorInfo represents error information in logs
type ErrorInfo struct {
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

// Global tracer provider and Zap logger
var (
	tracerProvider *trace.TracerProvider
	// Global service configuration for logging
	serviceConfig configs.Config
	// Global Zap logger instance
	globalLogger *ZapLogger
)

// InitializeOpenTelemetry sets up OpenTelemetry with proper exporters for ELK integration
// It configures OTLP HTTP exporter for production and stdout for development
func InitializeOpenTelemetry(cfg configs.Config) (func(), error) {
	ctx := context.Background()

	// Store service config globally for logging
	serviceConfig = cfg

	// Create resource with service information for ELK stack identification
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.OtelServiceName),
			semconv.ServiceVersionKey.String(cfg.OtelServiceVersion),
			semconv.DeploymentEnvironmentKey.String(cfg.OtelEnvironment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure trace exporter for ELK stack integration
	var exporter trace.SpanExporter
	if cfg.OtelOtlpEndpoint != "" {
		// Create OTLP gRPC exporter for ELK stack integration
		options := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OtelOtlpEndpoint),
			otlptracegrpc.WithTimeout(10 * time.Second), // Add connection timeout
		}

		// Add headers if configured (for authentication, custom routing, etc.)
		if len(cfg.OtelOtlpHeaders) > 0 {
			options = append(options, otlptracegrpc.WithHeaders(cfg.OtelOtlpHeaders))
		}

		// Use insecure for development, secure for production
		if cfg.OtelEnvironment == "development" || cfg.OtelEnvironment == "dev" {
			options = append(options, otlptracegrpc.WithInsecure())
		}

		exporter, err = otlptracegrpc.New(ctx, options...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
		}
	} else {
		// Use stdout exporter for development/testing
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	}

	// Create tracer provider with batch span processor for ELK stack
	batchOptions := []trace.BatchSpanProcessorOption{
		trace.WithBatchTimeout(time.Duration(cfg.OtelBatchTimeout) * time.Second),
		trace.WithExportTimeout(time.Duration(cfg.OtelExportInterval) * time.Second),
	}

	tracerProvider = trace.NewTracerProvider(
		trace.WithSpanProcessor(trace.NewBatchSpanProcessor(exporter, batchOptions...)),
		trace.WithResource(res),
		trace.WithSampler(trace.AlwaysSample()), // Use TraceIDRatioBased for production
	)

	// Set global tracer provider and propagator
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Initialize Zap logger
	if err := initializeZapLogger(cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize Zap logger: %w", err)
	}

	globalLogger.Info("OpenTelemetry and Zap initialized successfully",
		zap.String("service.name", cfg.OtelServiceName),
		zap.String("service.version", cfg.OtelServiceVersion),
		zap.String("environment", cfg.OtelEnvironment))

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			globalLogger.Error("Error shutting down tracer provider", zap.Error(err))
		}
	}, nil
}

// GetLogger returns the global Zap logger instance
func GetLogger() *ZapLogger {
	return globalLogger
}

// initializeZapLogger sets up Zap logger with production config for ELK stack
func initializeZapLogger(cfg configs.Config) error {
	// Configure Zap encoder for ELK stack
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create JSON encoder for structured logging
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// Configure log level
	var level zapcore.Level
	switch cfg.OtelEnvironment {
	case "development", "dev":
		level = zapcore.DebugLevel
	case "staging":
		level = zapcore.InfoLevel
	default: // production
		level = zapcore.WarnLevel
	}

	// Create core with console output (Docker/K8s will capture this)
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)

	// Create logger with caller info and stack trace for errors
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Add global fields for service identification
	logger = logger.With(
		zap.String("service.name", cfg.OtelServiceName),
		zap.String("service.version", cfg.OtelServiceVersion),
		zap.String("environment", cfg.OtelEnvironment),
	)

	// Create global logger instance
	globalLogger = &ZapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
	}

	return nil
}

// Info is a convenience function to log info messages using the global logger
func Info(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Info(msg, fields...)
	}
}

// Debug is a convenience function to log debug messages using the global logger
func Debug(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Debug(msg, fields...)
	}
}

// Warn is a convenience function to log warning messages using the global logger
func Warn(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Warn(msg, fields...)
	}
}

// Error is a convenience function to log error messages using the global logger
func Error(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Error(msg, fields...)
	}
}

// Fatal is a convenience function to log fatal messages and exit using the global logger
func Fatal(msg string, fields ...zap.Field) {
	if globalLogger != nil {
		globalLogger.Fatal(msg, fields...)
	}
}

// LogInfo creates a structured log entry and outputs it as JSON
func LogInfo(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	logStructured(ctx, LogLevelInfo, message, nil, attrs...)
}

// LogWarn creates a structured warning log entry
func LogWarn(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	logStructured(ctx, LogLevelWarn, message, nil, attrs...)
}

// LogError creates a structured error log entry
func LogError(ctx context.Context, err error, message string, attrs ...attribute.KeyValue) {
	var errorInfo *ErrorInfo
	if err != nil {
		errorInfo = &ErrorInfo{
			Message: err.Error(),
			Type:    fmt.Sprintf("%T", err),
		}
	}
	logStructured(ctx, LogLevelError, message, errorInfo, attrs...)
}

// LogDebug creates a structured debug log entry
func LogDebug(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	logStructured(ctx, LogLevelDebug, message, nil, attrs...)
}

// logStructured creates and outputs a structured log entry
func logStructured(ctx context.Context, level LogLevel, message string, errorInfo *ErrorInfo, attrs ...attribute.KeyValue) {
	// Extract trace information from context
	span := oteltrace.SpanFromContext(ctx)
	var traceID, spanID string
	if span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		spanID = span.SpanContext().SpanID().String()
	}

	// Convert OpenTelemetry attributes to map
	attributes := make(map[string]interface{})
	for _, attr := range attrs {
		attributes[string(attr.Key)] = attr.Value.AsInterface()
	}

	// Create structured log entry
	logEntry := StructuredLog{
		Timestamp:      time.Now().UTC(),
		Level:          level,
		Message:        message,
		ServiceName:    serviceConfig.OtelServiceName,
		ServiceVersion: serviceConfig.OtelServiceVersion,
		Environment:    serviceConfig.OtelEnvironment,
		TraceID:        traceID,
		SpanID:         spanID,
		Attributes:     attributes,
		Error:          errorInfo,
	}

	// Marshal to JSON and output using Zap
	if globalLogger != nil {
		// Use Zap for structured logging instead of fmt.Println
		switch level {
		case LogLevelDebug:
			globalLogger.Debug("Structured log entry",
				zap.String("trace.id", traceID),
				zap.String("span.id", spanID),
				zap.Any("attributes", attributes),
				zap.Any("error", errorInfo),
			)
		case LogLevelInfo:
			globalLogger.Info(message,
				zap.String("trace.id", traceID),
				zap.String("span.id", spanID),
				zap.Any("attributes", attributes),
			)
		case LogLevelWarn:
			globalLogger.Warn(message,
				zap.String("trace.id", traceID),
				zap.String("span.id", spanID),
				zap.Any("attributes", attributes),
			)
		case LogLevelError:
			fields := []zap.Field{
				zap.String("trace.id", traceID),
				zap.String("span.id", spanID),
				zap.Any("attributes", attributes),
			}
			if errorInfo != nil {
				fields = append(fields, zap.Any("error", errorInfo))
			}
			globalLogger.Error(message, fields...)
		}
	} else {
		// Fallback to JSON output for backward compatibility
		logJSON, err := json.Marshal(logEntry)
		if err != nil {
			// Use basic logging if Zap is not available
			fmt.Printf("Failed to marshal log entry: %v\n", err)
			return
		}
		fmt.Println(string(logJSON))
	}
}

// LogInfoWithTrace logs info with trace context
func LogInfoWithTrace(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	LogInfo(ctx, message, attrs...)
}

// LogErrorWithTrace logs error with trace context
func LogErrorWithTrace(ctx context.Context, err error, message string, attrs ...attribute.KeyValue) {
	LogError(ctx, err, message, attrs...)
}

// Simple, performance-focused helper functions for developers

// SpanError handles the common pattern: span.RecordError + LogError
func SpanError(ctx context.Context, span oteltrace.Span, err error, message string, attrs ...attribute.KeyValue) {
	if span != nil {
		span.RecordError(err)
	}
	LogError(ctx, err, message, attrs...)
}

// SpanInfo logs info with span attributes (optional)
func SpanInfo(ctx context.Context, span oteltrace.Span, message string, attrs ...attribute.KeyValue) {
	if span != nil && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	LogInfo(ctx, message, attrs...)
}

// SpanWarn logs warning with span attributes (optional)
func SpanWarn(ctx context.Context, span oteltrace.Span, message string, attrs ...attribute.KeyValue) {
	if span != nil && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	LogWarn(ctx, message, attrs...)
}

// SpanDebug logs debug with span attributes (optional)
func SpanDebug(ctx context.Context, span oteltrace.Span, message string, attrs ...attribute.KeyValue) {
	if span != nil && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	LogDebug(ctx, message, attrs...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil && globalLogger.logger != nil {
		return globalLogger.logger.Sync()
	}
	return nil
}
