package services

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	auditdtos "go-boilerplate/internal/dtos/audit_dtos"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/utils/logs"
	"time"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type AuditService interface {
	PublishAuditLog(ctx context.Context, dto auditdtos.CreateAuditLogDTO) error
	ProcessIncomingAuditLog(ctx context.Context, msg auditdtos.AuditLogMessage) error
}

type auditService struct {
	producer messaging.Producer
	cfg      configs.Config
	tracer   trace.Tracer
	v        *validator.Validate
}

// NewAuditService creates a new AuditService
func NewAuditService(
	producer messaging.Producer,
	cfg configs.Config,
	v *validator.Validate,
) AuditService {
	return &auditService{
		producer: producer,
		cfg:      cfg,
		tracer:   otel.Tracer("audit-service"),
		v:        v,
	}
}

func (s *auditService) PublishAuditLog(ctx context.Context, dto auditdtos.CreateAuditLogDTO) error {
	ctx, span := s.tracer.Start(ctx, "AuditService.PublishAuditLog")
	defer span.End()

	if err := s.v.Struct(dto); err != nil {
		logs.SpanError(ctx, span, err, "Validation failed for audit log")
		return fmt.Errorf("validation failed: %w", err)
	}

	traceID := ""
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		traceID = spanContext.TraceID().String()
	}

	msg := auditdtos.AuditLogMessage{
		CreateAuditLogDTO: dto,
		Timestamp:         time.Now(),
		TraceID:           traceID,
	}

	// Publish to RabbitMQ using default exchange ("") and routing key as queue name
	err := s.producer.PublishJSON(ctx, "", "audit_logs_queue", msg)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to publish audit log")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	logs.SpanInfo(ctx, span, "Audit log published successfully")
	return nil
}

func (s *auditService) ProcessIncomingAuditLog(ctx context.Context, msg auditdtos.AuditLogMessage) error {
	ctx, span := s.tracer.Start(ctx, "AuditService.ProcessIncomingAuditLog")
	defer span.End()

	// Simulate processing the message (e.g. saving to DB)
	logs.SpanInfo(ctx, span, fmt.Sprintf("Processing Audit Log -> Action: %s, Entity: %s, ID: %s, TraceID: %s", 
		msg.Action, msg.Entity, msg.EntityID, msg.TraceID))

	// If we were saving to DB, we'd do it here.

	return nil
}
