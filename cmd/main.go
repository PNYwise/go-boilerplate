package main

import (
	"context"
	"flag"
	"fmt"
	"go-boilerplate/internal/apps"
	"go-boilerplate/internal/configs"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

func main() {
	// Parse command-line arguments
	modeFlag := flag.String("mode", "", "application mode: http, grpc, rabbit")
	stageFlag := flag.String("stage", "", "stage name: dev, staging, prod, etc.")
	flag.Parse()

	// Determine mode (CLI flag -> ENV var -> default)
	mode := strings.ToLower(strings.TrimSpace(*modeFlag))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(os.Getenv("MODE")))
	}
	if mode == "" {
		mode = "http" // default
	}

	// Determine stage (CLI flag -> ENV var)
	stage := strings.TrimSpace(*stageFlag)
	if stage == "" {
		stage = strings.TrimSpace(os.Getenv("STAGE"))
	}

	// Load configuration
	cfg := configs.MustLoad(mode, stage)

	// Create application instance with all dependencies wired
	app, err := apps.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create application: %v\n", err)
		os.Exit(1)
	}
	defer app.Shutdown()

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Convert string mode to apps.Mode
	var appMode apps.Mode
	switch mode {
	case "http":
		appMode = apps.ModeHTTP
	case "grpc":
		appMode = apps.ModeGRPC
	case "rabbit":
		appMode = apps.ModeRabbit
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		os.Exit(1)
	}

	// Run the application
	fmt.Printf("🚀 Starting application in %s mode\n", mode)
	if err := app.Run(ctx, appMode); err != nil {
		app.GetLogger().Error("Application failed",
			zap.String("mode", mode),
			zap.Error(err))
		os.Exit(1)
	}
}
