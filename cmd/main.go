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

	_ "github.com/go-sql-driver/mysql" // Database driver
)

// main bootstraps the application - DO NOT MODIFY
// All customization should happen in internal/apps/wire.go
func main() {
	defer handlePanic()

	cfg := loadConfig()
	app := createApp(cfg)
	defer app.Shutdown()

	runApp(app)
}

// handlePanic provides top-level panic recovery - DO NOT MODIFY
func handlePanic() {
	if r := recover(); r != nil {
		fmt.Printf("Application crashed with panic: %v\n", r)
		os.Exit(1)
	}
}

// loadConfig loads application configuration - DO NOT MODIFY
func loadConfig() configs.Config {
	stageFlag := flag.String("stage", "", "stage name: dev, staging, prod, etc.")
	flag.Parse()

	stage := getStage(*stageFlag)

	// Seamless - no mode needed, just load config
	return configs.MustLoad(stage)
}

// createApp creates the application instance - DO NOT MODIFY
func createApp(cfg configs.Config) *apps.App {
	app, err := apps.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create application: %v\n", err)
		os.Exit(1)
	}
	return app
}

// runApp runs the application with graceful shutdown - DO NOT MODIFY
func runApp(app *apps.App) {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	defer cancel()

	fmt.Println("Starting HTTP server")
	fmt.Println("Press Ctrl+C to stop gracefully")

	if err := app.Run(ctx); err != nil {
		fmt.Printf("Application failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Application stopped")
}

// Helper functions - DO NOT MODIFY

func getStage(flag string) string {
	stage := strings.TrimSpace(flag)
	if stage == "" {
		stage = strings.TrimSpace(os.Getenv("STAGE"))
	}
	return stage
}
