package apps

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"

	"go.uber.org/zap"
)

// App represents the core application - DO NOT MODIFY
// This is infrastructure code that handles application lifecycle
type App struct {
	application *Application
	cleanup     func()
	logger      *zap.Logger
	config      configs.Config
}

// New creates a new App instance - DO NOT MODIFY
// Use wire.go to customize application dependencies
func New(cfg configs.Config) (*App, error) {
	application, cleanup, err := InitializeApp(cfg)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("failed to initialize application: %w", err)
	}

	return &App{
		application: application,
		cleanup:     cleanup,
		logger:      application.Logger,
		config:      cfg,
	}, nil
}

// Run starts the application - DO NOT MODIFY
func (a *App) Run(ctx context.Context) (err error) {
	// Panic recovery to ensure cleanup happens
	defer func() {
		if r := recover(); r != nil {
			if a.logger != nil {
				a.logger.Error("Application panicked", zap.Any("panic", r))
			}
			a.ShutdownWithPanic()
			err = fmt.Errorf("application panicked: %v", r)
		}
	}()

	a.logger.Info("Starting application server", zap.String("address", a.config.HTTPAddr))

	return a.application.Server.Run(ctx, a.config.HTTPAddr)
}

// Shutdown gracefully shuts down the application - DO NOT MODIFY
func (a *App) Shutdown() {
	a.shutdown(false)
}

// ShutdownWithPanic handles shutdown after a panic - DO NOT MODIFY
func (a *App) ShutdownWithPanic() {
	a.shutdown(true)
}

// GetLogger returns the application logger - DO NOT MODIFY
func (a *App) GetLogger() *zap.Logger {
	return a.logger
}

// shutdown handles the actual shutdown process - DO NOT MODIFY
func (a *App) shutdown(fromPanic bool) {
	if fromPanic {
		if a.logger != nil {
			a.logger.Error("Application shutting down due to panic, cleaning up resources")
		}
	} else {
		if a.logger != nil {
			a.logger.Info("Application shutting down gracefully")
		}
	}

	// Always attempt cleanup, even if one fails
	defer func() {
		if r := recover(); r != nil && a.logger != nil {
			a.logger.Error("Panic during shutdown cleanup", zap.Any("panic", r))
		}
	}()

	if a.cleanup != nil {
		func() {
			defer func() {
				if r := recover(); r != nil && a.logger != nil {
					a.logger.Error("Panic during cleanup function", zap.Any("panic", r))
				}
			}()
			a.cleanup()
		}()
	}

	// Shutdown server gracefully
	if a.application != nil && a.application.Server != nil {
		func() {
			defer func() {
				if r := recover(); r != nil && a.logger != nil {
					a.logger.Error("Panic during server shutdown", zap.Any("panic", r))
				}
			}()
			// Server shutdown handled by context cancellation in Run method
		}()
	}

	if a.logger != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Can't log this since logger sync failed
				}
			}()
			_ = a.logger.Sync()
		}()
	}
}
