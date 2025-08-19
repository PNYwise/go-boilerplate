package main

import (
	"context"
	"flag"
	"fmt"
	app "go-boilerplate/internal/apps"
	"go-boilerplate/internal/configs"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// how to run:
	// go run cmd/main.go --mode http --stage dev
	// go run cmd/main.go --mode rabbit --stage dev

	// how to build:
	// go build -o bin/app cmd/main.go
	// ./bin/app --mode http --stage dev


	modeFlag := flag.String("mode", "", "application mode: http or rabbit")
	stageFlag := flag.String("stage", "", "stage name: dev, staging, prod, etc.")
	flag.Parse()

	mode := strings.ToLower(strings.TrimSpace(*modeFlag))
	if mode == "" {
		// fallback to env if not supplied
		mode = strings.ToLower(strings.TrimSpace(os.Getenv("MODE")))
	}
	if mode == "" {
		mode = "http" // sane default
	}

	stage := strings.TrimSpace(*stageFlag)
	if stage == "" {
		stage = strings.TrimSpace(os.Getenv("STAGE"))
	}
	cfg := configs.MustLoad(mode, stage)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a, err := app.New(cfg)
	if err != nil {
		panic(err)
	}
	if err := a.Run(ctx, app.Mode(mode)); err != nil {
		panic(err)
	}
}
