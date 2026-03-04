package apps

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"

	"go.uber.org/zap"
)

// Mode represents the application mode in which it runs.
// It can be HTTP, RabbitMQ, or gRPC.
type Mode string

// Available application modes.
const (
	// ModeHTTP represents the HTTP server mode.
	ModeHTTP Mode = "http"
	// ModeRabbit represents the RabbitMQ consumer mode.
	ModeRabbit Mode = "rabbit"
	// ModeGRPC represents the gRPC server mode.
	ModeGRPC Mode = "grpc"
)

// App represents the main application with all wired dependencies
type App struct {
	application *Application
	cleanup     func()
	logger      *zap.Logger
	config      configs.Config
}

// New creates a new App instance with all dependencies wired via Wire
func New(cfg configs.Config) (*App, error) {
	// Use Wire to initialize all dependencies
	application, cleanup, err := InitializeApp(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize application: %w", err)
	}

	return &App{
		application: application,
		cleanup:     cleanup,
		logger:      application.Logger,
		config:      cfg,
	}, nil
}

// Run starts the application in the specified mode
func (a *App) Run(ctx context.Context, mode Mode) error {
	a.logger.Info("Application starting", zap.String("mode", string(mode)))

	switch mode {
	case ModeHTTP:
		a.logger.Info("Starting HTTP server", zap.String("address", a.config.HTTPAddr))
		return a.application.HTTPServer.Run(ctx, a.config.HTTPAddr)
	case ModeGRPC:
		return fmt.Errorf("gRPC mode not implemented yet")
	case ModeRabbit:
		return fmt.Errorf("RabbitMQ mode not implemented yet")
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() {
	if a.cleanup != nil {
		a.cleanup()
	}
	if a.logger != nil {
		_ = a.logger.Sync()
	}
}

// GetLogger returns the application logger
func (a *App) GetLogger() *zap.Logger {
	return a.logger
}