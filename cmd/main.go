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
	// mode := os.Getenv("MODE") // http, rabbit, gr
	// 	// ---- 1) Parse simple CLI flags (no binding) ----
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

	fmt.Println("Starting application in mode: ", mode)
	fmt.Println("Application Name: ", cfg.AppName)
	fmt.Println("HTTP Address: ", cfg.HTTPAddr)
	fmt.Println("db Address: ", cfg.DbHost)
	fmt.Println("db Address: ", cfg.DbName)

	a, err := app.New(cfg)
	if err != nil {
		panic(err)
	}
	if err := a.Run(ctx, app.Mode(mode)); err != nil {
		panic(err)
	}
}
