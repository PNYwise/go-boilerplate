package services

import (
	"context"
	"database/sql"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/utils/logs"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
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
	db     *sql.DB
	redis  *redis.Client
	rabbit *messaging.RabbitMQConnection
}

// HealthServiceParams holds dependencies for the HealthService
type HealthServiceParams struct {
	fx.In

	Cfg    configs.Config
	V      *validator.Validate
	Db     *sql.DB                       `optional:"true"`
	Redis  *redis.Client                 `optional:"true"`
	Rabbit *messaging.RabbitMQConnection `optional:"true"`
}

// NewHealthService creates a new health service instance with OpenTelemetry instrumentation
func NewHealthService(params HealthServiceParams) HealthService {
	return &healthService{
		cfg:    params.Cfg,
		tracer: otel.Tracer("health-service"),
		v:      params.V,
		db:     params.Db,
		redis:  params.Redis,
		rabbit: params.Rabbit,
	}
}

func (s *healthService) Check(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "HealthService.Check")
	defer span.End()

	logs.SpanInfo(ctx, span, "Health check performed")
	return nil
}

func (s *healthService) GetStatus(ctx context.Context) map[string]interface{} {
	ctx, span := s.tracer.Start(ctx, "HealthService.GetStatus")
	defer span.End()

	status := map[string]interface{}{
		"app":  s.cfg.AppName,
		"http": "healthy",
	}

	// Check Database
	if s.db != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.db.PingContext(dbCtx); err != nil {
			status["database"] = "unhealthy"
			logs.SpanError(ctx, span, err, "Database health check failed")
		} else {
			status["database"] = "healthy"
		}
	} else {
		status["database"] = "not configured"
	}

	// Check Redis
	if s.redis != nil {
		redisCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := s.redis.Ping(redisCtx).Err(); err != nil {
			status["redis"] = "unhealthy"
			logs.SpanError(ctx, span, err, "Redis health check failed")
		} else {
			status["redis"] = "healthy"
		}
	} else {
		status["redis"] = "not configured"
	}

	// Check RabbitMQ
	if s.rabbit != nil {
		if s.rabbit.IsConnected() {
			status["rabbitmq"] = "healthy"
		} else {
			status["rabbitmq"] = "unhealthy"
		}
	} else {
		status["rabbitmq"] = "not configured"
	}

	return status
}
