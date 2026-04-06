// Package logs provides OpenTelemetry-based observability for the go-boilerplate application
// following best practices for distributed tracing and logging with ELK stack integration
package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/configs"
	stdlog "log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	ServiceNameDB = "database"
)

// LogLevel represents different log levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// StructuredLog represents a structured log entry compatible with ELK stack
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

// ErrorInfo represents error information in logs
type ErrorInfo struct {
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

// Global tracer provider and tracers
var (
	tracerProvider *trace.TracerProvider
	appTracer      oteltrace.Tracer
	userSvcTracer  oteltrace.Tracer
	healthTracer   oteltrace.Tracer
	repoTracer     oteltrace.Tracer
	httpTracer     oteltrace.Tracer
	dbTracer       oteltrace.Tracer
	// Global service configuration for logging
	serviceConfig configs.Config
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

	// Initialize tracers for different components
	appTracer = otel.Tracer("app")
	repoTracer = otel.Tracer("repository")
	httpTracer = otel.Tracer("http")
	dbTracer = otel.Tracer("database")

	stdlog.Printf("OpenTelemetry initialized successfully with service name: %s", cfg.OtelServiceName)

	// Return cleanup function
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerProvider.Shutdown(ctx); err != nil {
			stdlog.Printf("Error shutting down tracer provider: %v", err)
		}
	}, nil
}

// GetDBTracer returns the database tracer
func GetDBTracer() oteltrace.Tracer {
	return dbTracer
}

// GetUserServiceTracer returns the user service tracer
func GetUserServiceTracer() oteltrace.Tracer {
	return userSvcTracer
}

// GetHTTPTracer returns the HTTP tracer
func GetHTTPTracer() oteltrace.Tracer {
	return httpTracer
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

	// Marshal to JSON and output
	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		stdlog.Printf("Failed to marshal log entry: %v", err)
		return
	}

	// Output to stdout (will be captured by Docker/Kubernetes and sent to ELK)
	fmt.Println(string(logJSON))
}

// LogInfoWithTracer creates a span and logs an info event with attributes
func LogInfoWithTracer(ctx context.Context, tracer oteltrace.Tracer, operation string, message string, attrs ...attribute.KeyValue) {
	ctx, span := tracer.Start(ctx, operation)
	defer span.End()

	span.SetAttributes(attrs...)
	span.AddEvent(message, oteltrace.WithAttributes(attrs...))
	span.SetStatus(codes.Ok, "")

	// Also log as structured JSON
	LogInfo(ctx, message, attrs...)
}

// LogErrorWithTracer creates a span and logs an error event with attributes
func LogErrorWithTracer(ctx context.Context, tracer oteltrace.Tracer, operation string, err error, message string, attrs ...attribute.KeyValue) {
	ctx, span := tracer.Start(ctx, operation)
	defer span.End()

	span.SetAttributes(attrs...)
	span.RecordError(err)
	span.AddEvent(message, oteltrace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, err.Error())

	// Also log as structured JSON
	LogError(ctx, err, message, attrs...)
}

// LogDBOperation logs database operations with query information
func LogDBOperation(ctx context.Context, operation, query string, args []interface{}, duration time.Duration, err error) {
	ctx, span := dbTracer.Start(ctx, operation)
	defer span.End()

	attrs := []attribute.KeyValue{
		semconv.DBStatementKey.String(query),
		semconv.DBOperationKey.String(operation),
		attribute.String("db.query.duration", duration.String()),
	}

	if len(args) > 0 {
		attrs = append(attrs, attribute.Int("db.query.args_count", len(args)))
	}

	span.SetAttributes(attrs...)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		LogError(ctx, err, fmt.Sprintf("Database operation %s failed", operation), attrs...)
	} else {
		span.SetStatus(codes.Ok, "")
		LogInfo(ctx, fmt.Sprintf("Database operation %s completed", operation), attrs...)
	}
}

// LogHTTPRequest logs HTTP request information
func LogHTTPRequest(ctx context.Context, method, path string, statusCode int, requestSize, responseSize int64, duration time.Duration) {
	ctx, span := httpTracer.Start(ctx, fmt.Sprintf("%s %s", method, path))
	defer span.End()

	attrs := []attribute.KeyValue{
		semconv.HTTPMethodKey.String(method),
		semconv.HTTPRouteKey.String(path),
		semconv.HTTPStatusCodeKey.Int(statusCode),
		attribute.Int64("http.request.size", requestSize),
		attribute.Int64("http.response.size", responseSize),
		attribute.String("http.request.duration", duration.String()),
	}

	span.SetAttributes(attrs...)

	if statusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
		LogWarn(ctx, fmt.Sprintf("HTTP request failed: %s %s", method, path), attrs...)
	} else {
		span.SetStatus(codes.Ok, "")
		LogInfo(ctx, fmt.Sprintf("HTTP request completed: %s %s", method, path), attrs...)
	}
}

// Utility functions for environment detection
func isProduction() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "production" || env == "prod"
}

func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	return "development"
}

func getOTLPEndpoint(cfg configs.Config) string {
	// Check for OTLP endpoint in environment variables or config
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	// Default to OpenTelemetry Collector endpoint
	return "http://localhost:4318/v1/traces"
}
