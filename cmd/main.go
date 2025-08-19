package main

import (
	"context"
	app "go-boilerplate/internal/apps"
	"go-boilerplate/internal/configs"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	mode := os.Getenv("MODE") // http, rabbit, grpc
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load your application configuration
	cfg := configs.MustLoad()

	a, err := app.New(cfg)
	if err != nil {
		panic(err)
	}
	if err := a.Run(ctx, app.Mode(mode)); err != nil {
		panic(err)
	}
}
