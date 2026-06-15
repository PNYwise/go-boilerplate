package handlers

import (
	"context"
	"encoding/json"
	auditdtos "go-boilerplate/internal/dtos/audit_dtos"
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/utils/logs"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type AuditWorker struct {
	auditSrv services.AuditService
	tracer   trace.Tracer
}

func NewAuditWorker(auditSrv services.AuditService) *AuditWorker {
	return &AuditWorker{
		auditSrv: auditSrv,
		tracer:   otel.Tracer("audit-worker"),
	}
}

// HandleAuditLog processes messages from the audit log queue
func (h *AuditWorker) HandleAuditLog(ctx context.Context, msg amqp.Delivery) error {
	// Extract OpenTelemetry context from AMQP headers
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, headersCarrier(msg.Headers))

	ctx, span := h.tracer.Start(ctx, "AuditWorker.HandleAuditLog")
	defer span.End()

	var payload auditdtos.AuditLogMessage
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		logs.SpanError(ctx, span, err, "Failed to unmarshal audit log message")
		return err // Returning error causes Nack
	}

	logs.SpanInfo(ctx, span, "Successfully parsed audit log message")

	// Call the service to process the business logic
	if err := h.auditSrv.ProcessIncomingAuditLog(ctx, payload); err != nil {
		logs.SpanError(ctx, span, err, "AuditService failed to process incoming log")
		return err
	}

	return nil
}

// headersCarrier adapts amqp.Table to satisfy the propagation.TextMapCarrier interface
// We duplicate this here to extract exactly how the producer injected it.
type headersCarrier amqp.Table

func (c headersCarrier) Get(key string) string {
	val, ok := c[key]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return ""
	}
	return strVal
}

func (c headersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c headersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
