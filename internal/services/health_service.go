package services

import (
	"context"
	"go-boilerplate/internal/configs"

	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// HealthService defines health check operations
type HealthService interface {
	Check(ctx context.Context) error
	GetStatus(ctx context.Context) map[string]interface{}
}

type healthService struct {
	cfg    configs.Config
	tracer trace.Tracer // OpenTelemetry tracer for health service
	v      *validator.Validate
}

// NewHealthService creates a new health service instance with OpenTelemetry instrumentation
func NewHealthService(
	cfg configs.Config,
	v *validator.Validate,
) HealthService {
	return &healthService{
		cfg:    cfg,
		tracer: otel.Tracer("health-service"),
		v:      v,
	}
}

func (s *healthService) Check(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "HealthService.Check")
	defer span.End()

	span.SetStatus(codes.Ok, "health check performed")
	span.AddEvent("Health check performed")
	return nil
}

func (s *healthService) GetStatus(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"status": "healthy",
		"app":    s.cfg.AppName,
	}
}
