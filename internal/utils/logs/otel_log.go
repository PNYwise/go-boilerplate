// Package logs provides OpenTelemetry-based observability with Zap structured logging
// following best practices for distributed tracing and high-performance logging with ELK stack integration
package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/configs"
	"os"
	"strings"
	"time"

	goruntime "runtime"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel represents log level enumeration
type LogLevel string

type traceIDKeyType struct{}
type spanIDKeyType struct{}
type startTimeKeyType struct{}

// Keys to store trace and span IDs in context when OpenTelemetry is disabled
var TraceIDKey = traceIDKeyType{}
var SpanIDKey = spanIDKeyType{}
var StartTimeKey = startTimeKeyType{}

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// StructuredLog represents the structure for JSON logs
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

type ErrorInfo struct {
	Message    string `json:"message"`
	Type       string `json:"type,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

// ZapLogger wraps Zap logger with OpenTelemetry integration
type ZapLogger struct {
	logger *zap.Logger
	skip1  *zap.Logger
}

func (z *ZapLogger) Info(msg string, fields ...zap.Field)  { z.skip1.Info(msg, fields...) }
func (z *ZapLogger) Debug(msg string, fields ...zap.Field) { z.skip1.Debug(msg, fields...) }
func (z *ZapLogger) Warn(msg string, fields ...zap.Field)  { z.skip1.Warn(msg, fields...) }
func (z *ZapLogger) Error(msg string, fields ...zap.Field) { z.skip1.Error(msg, fields...) }
func (z *ZapLogger) Fatal(msg string, fields ...zap.Field) { z.skip1.Fatal(msg, fields...) }

// With adds structured context to the logger
func (z *ZapLogger) With(fields ...zap.Field) *ZapLogger {
	newLogger := z.logger.With(fields...)
	return &ZapLogger{
		logger: newLogger,
		skip1:  newLogger.WithOptions(zap.AddCallerSkip(1)),
	}
}

var (
	tracerProvider *sdktrace.TracerProvider
	serviceConfig  configs.Config
	globalLogger   *ZapLogger
	skip1Logger    *zap.Logger
	skip2Logger    *zap.Logger
)

// InitializeOpenTelemetry sets up OpenTelemetry. Only sends to OTLP endpoint (high performance).
func InitializeOpenTelemetry(cfg configs.Config) (func(), error) {
	ctx := context.Background()
	serviceConfig = cfg

	if !cfg.OtelEnabled {
		if err := initializeZapLogger(cfg, nil, nil); err != nil {
			return nil, err
		}
		globalLogger.Info("Running WITHOUT OpenTelemetry (OTEL_ENABLED is false)")
		return func() {}, nil
	}

	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.AppName),
			semconv.ServiceVersion(cfg.OtelServiceVersion),
			attribute.String("service.node.name", cfg.OtelServiceNodeName),
			attribute.String("deployment.environment", cfg.OtelEnvironment),
			attribute.String("telemetry.sdk.language", "go"),
			attribute.String("service.runtime.version", goruntime.Version()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	if strings.TrimSpace(cfg.OtelOtlpEndpoint) == "" {
		if err := initializeZapLogger(cfg, nil, nil); err != nil {
			return nil, err
		}
		globalLogger.Info("Running WITHOUT OpenTelemetry (OTLP Endpoint empty)")
		return func() {}, nil
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(cfg.OtelOtlpEndpoint), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("build trace exporter: %w", err)
	}

	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithEndpoint(cfg.OtelOtlpEndpoint), otlploggrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("build log exporter: %w", err)
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpoint(cfg.OtelOtlpEndpoint), otlpmetricgrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("build metric exporter: %w", err)
	}

	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	metricProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(metricProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	if err := otelruntime.Start(otelruntime.WithMeterProvider(metricProvider)); err != nil {
		return nil, fmt.Errorf("start runtime metrics: %w", err)
	}

	if err := initializeZapLogger(cfg, res, logExporter); err != nil {
		return nil, fmt.Errorf("failed to initialize Zap logger: %w", err)
	}

	globalLogger.Info("OpenTelemetry initialized (sending data to Collector only)")

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if tracerProvider != nil {
			_ = tracerProvider.Shutdown(ctx)
		}
	}, nil
}

func GetLogger() *ZapLogger {
	return globalLogger
}

func initializeZapLogger(cfg configs.Config, res *resource.Resource, logExporter sdklog.Exporter) error {
	level := zapcore.WarnLevel
	switch cfg.OtelEnvironment {
	case "development", "dev":
		level = zapcore.DebugLevel
	case "staging":
		level = zapcore.InfoLevel
	}

	if cfg.LogLevel != "" {
		switch strings.ToLower(cfg.LogLevel) {
		case "debug":
			level = zapcore.DebugLevel
		case "info":
			level = zapcore.InfoLevel
		case "warn", "warning":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "@timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	stdoutCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(os.Stdout), level)
	var cores []zapcore.Core = []zapcore.Core{stdoutCore}

	if logExporter != nil && res != nil {
		logProvider := sdklog.NewLoggerProvider(
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
			sdklog.WithResource(res),
		)
		otelCore := otelzap.NewCore(cfg.AppName, otelzap.WithLoggerProvider(logProvider))
		cores = append(cores, otelCore)
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)).With(
		zap.String("service.name", cfg.AppName),
		zap.String("service.version", cfg.OtelServiceVersion),
		zap.String("environment", cfg.OtelEnvironment),
	)

	globalLogger = &ZapLogger{
		logger: logger,
		skip1:  logger.WithOptions(zap.AddCallerSkip(1)),
	}
	skip1Logger = globalLogger.skip1
	skip2Logger = logger.WithOptions(zap.AddCallerSkip(2))

	return nil
}

// Convenience global logging functions
func Info(msg string, fields ...zap.Field)  { if skip1Logger != nil { skip1Logger.Info(msg, fields...) } }
func Debug(msg string, fields ...zap.Field) { if skip1Logger != nil { skip1Logger.Debug(msg, fields...) } }
func Warn(msg string, fields ...zap.Field)  { if skip1Logger != nil { skip1Logger.Warn(msg, fields...) } }
func Error(msg string, fields ...zap.Field) { if skip1Logger != nil { skip1Logger.Error(msg, fields...) } }
func Fatal(msg string, fields ...zap.Field) { if skip1Logger != nil { skip1Logger.Fatal(msg, fields...) } }

// LogInfo creates a structured log entry and outputs it as JSON
func LogInfo(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	logStructured(ctx, LogLevelInfo, message, nil, attrs...)
}

func LogWarn(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	logStructured(ctx, LogLevelWarn, message, nil, attrs...)
}

func LogError(ctx context.Context, err error, message string, attrs ...attribute.KeyValue) {
	var errorInfo *ErrorInfo
	if err != nil {
		errorInfo = &ErrorInfo{Message: err.Error(), Type: fmt.Sprintf("%T", err)}
	}
	logStructured(ctx, LogLevelError, message, errorInfo, attrs...)
}

func LogDebug(ctx context.Context, message string, attrs ...attribute.KeyValue) {
	logStructured(ctx, LogLevelDebug, message, nil, attrs...)
}

func logStructured(ctx context.Context, level LogLevel, message string, errorInfo *ErrorInfo, attrs ...attribute.KeyValue) {
	span := oteltrace.SpanFromContext(ctx)
	var traceID, spanID string
	if span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		spanID = span.SpanContext().SpanID().String()
	}

	if traceID == "" {
		if tid, ok := ctx.Value(TraceIDKey).(string); ok && tid != "" {
			traceID = tid
		}
	}
	if spanID == "" {
		if sid, ok := ctx.Value(SpanIDKey).(string); ok && sid != "" {
			spanID = sid
		}
	}

	attributes := make(map[string]interface{}, len(attrs))
	for _, attr := range attrs {
		attributes[string(attr.Key)] = attr.Value.AsInterface()
	}

	// Automatically calculate and inject http.duration for every log if StartTimeKey is present
	if start, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		if _, exists := attributes["http.duration"]; !exists {
			attributes["http.duration"] = time.Since(start).String()
		}
	}

	if globalLogger != nil {
		fields := []zap.Field{
			zap.String("trace.id", traceID),
			zap.String("span.id", spanID),
			zap.Any("attributes", attributes),
		}

		if errorInfo != nil {
			fields = append(fields, zap.Any("error", errorInfo))
		}

		switch level {
		case LogLevelDebug:
			skip2Logger.Debug(message, fields...)
		case LogLevelInfo:
			skip2Logger.Info(message, fields...)
		case LogLevelWarn:
			skip2Logger.Warn(message, fields...)
		case LogLevelError:
			skip2Logger.Error(message, fields...)
		}
	} else {
		// Absolute Fallback
		logEntry := StructuredLog{
			Timestamp:   time.Now().UTC(),
			Level:       level,
			Message:     message,
			TraceID:     traceID,
			SpanID:      spanID,
			Attributes:  attributes,
			Error:       errorInfo,
		}
		if logJSON, err := json.Marshal(logEntry); err == nil {
			fmt.Println(string(logJSON))
		}
	}
}

// SpanError handles the common pattern: span.RecordError + LogError
func SpanError(ctx context.Context, span oteltrace.Span, err error, message string, attrs ...attribute.KeyValue) {
	if span != nil {
		span.RecordError(err)
	}
	var errorInfo *ErrorInfo
	if err != nil {
		errorInfo = &ErrorInfo{Message: err.Error(), Type: fmt.Sprintf("%T", err)}
	}
	logStructured(ctx, LogLevelError, message, errorInfo, attrs...)
}

func SpanInfo(ctx context.Context, span oteltrace.Span, message string, attrs ...attribute.KeyValue) {
	if span != nil && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	logStructured(ctx, LogLevelInfo, message, nil, attrs...)
}

// SpanWarn logs warning with span attributes (optional)
func SpanWarn(ctx context.Context, span oteltrace.Span, message string, attrs ...attribute.KeyValue) {
	if span != nil && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	logStructured(ctx, LogLevelWarn, message, nil, attrs...)
}

// SpanDebug logs debug with span attributes (optional)
func SpanDebug(ctx context.Context, span oteltrace.Span, message string, attrs ...attribute.KeyValue) {
	if span != nil && len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	logStructured(ctx, LogLevelDebug, message, nil, attrs...)
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil && globalLogger.logger != nil {
		return globalLogger.logger.Sync()
	}
	return nil
}
