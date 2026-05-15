package apps

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/utils/logs"
)

// App represents the core application - DO NOT MODIFY
// This is infrastructure code that handles application lifecycle
type App struct {
	application *Application
	cleanup     func()
	config      configs.Config
}

// New creates a new App instance - DO NOT MODIFY
// Use wire.go to customize application dependencies
func New(cfg configs.Config) (*App, error) {
	// Initialize OpenTelemetry first
	otelCleanup, err := logs.InitializeOpenTelemetry(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}

	application, cleanup, err := InitializeApp(cfg)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		otelCleanup() // Clean up OpenTelemetry on failure
		return nil, fmt.Errorf("failed to initialize application: %w", err)
	}

	// Combine cleanup functions
	combinedCleanup := func() {
		if cleanup != nil {
			cleanup()
		}
		otelCleanup()
	}

	return &App{
		application: application,
		cleanup:     combinedCleanup,
		config:      cfg,
	}, nil
}

// Run starts the application - DO NOT MODIFY
func (a *App) Run(ctx context.Context) error {
	// Panic recovery to ensure cleanup happens
	defer func() {
		if r := recover(); r != nil {
			a.ShutdownWithPanic()
		}
	}()

	return a.application.Server.Run(ctx, a.config.HTTPAddr)
}

// Shutdown gracefully shuts down the application - DO NOT MODIFY
func (a *App) Shutdown() {
	a.shutdown()
}

// ShutdownWithPanic handles shutdown after a panic - DO NOT MODIFY
func (a *App) ShutdownWithPanic() {
	a.shutdown()
}

// shutdown handles the actual shutdown process - DO NOT MODIFY
func (a *App) shutdown() {
	// Always attempt cleanup, even if one fails
	defer func() {
		if r := recover(); r != nil {
			// Handle panic during shutdown
		}
	}()

	if a.cleanup != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic during cleanup
				}
			}()
			a.cleanup()
		}()
	}

	// Shutdown server gracefully
	if a.application != nil && a.application.Server != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic during server shutdown
				}
			}()
			// Server shutdown handled by context cancellation in Run method
		}()
	}
}
