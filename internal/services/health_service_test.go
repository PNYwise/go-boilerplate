package services_test

import (
	"context"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/services"
	"testing"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

func TestHealthService(t *testing.T) {
	// Example of manual dependency injection for testing
	cfg := configs.Config{
		AppName: "test-app",
	}

	logger, _ := zap.NewDevelopment()
	validator := validator.New()

	healthService := services.NewHealthService(cfg, logger, validator)

	// Test Check method
	ctx := context.Background()
	err := healthService.Check(ctx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test GetStatus method
	status := healthService.GetStatus(ctx)
	if status["status"] != "healthy" {
		t.Errorf("Expected status to be 'healthy', got: %v", status["status"])
	}

	if status["app"] != "test-app" {
		t.Errorf("Expected app to be 'test-app', got: %v", status["app"])
	}
}

// Example of how you would use Wire for more complex testing scenarios
// func TestWithWire(t *testing.T) {
//     cfg := configs.Config{AppName: "test-app"}
//
//     testApp, err := wire.InitializeTestApp(cfg)
//     if err != nil {
//         t.Fatal(err)
//     }
//
//     // Use testApp.Services.HealthService for testing
//     err = testApp.Services.HealthService.Check(context.Background())
//     if err != nil {
//         t.Errorf("Health check failed: %v", err)
//     }
// }
