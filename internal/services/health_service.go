package services

import (
	"context"
	"go-boilerplate/internal/configs"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// HealthService defines health check operations
type HealthService interface {
	Check(ctx context.Context) error
	GetStatus(ctx context.Context) map[string]interface{}
}

type healthService struct {
	cfg    configs.Config
	logger *zap.Logger
	v      *validator.Validate
}

// NewHealthService creates a new health service instance
func NewHealthService(
	cfg configs.Config,
	logger *zap.Logger,
	v *validator.Validate,
) HealthService {
	return &healthService{
		cfg:    cfg,
		logger: logger,
		v:      v,
	}
}

func (s *healthService) Check(ctx context.Context) error {
	s.logger.Info("Health check performed")
	return nil
}

func (s *healthService) GetStatus(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"status": "healthy",
		"app":    s.cfg.AppName,
	}
}