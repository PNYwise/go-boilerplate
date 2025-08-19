package app

import (
	"context"
	"fmt"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/dbs"
	"go-boilerplate/internal/logs"
	"go-boilerplate/internal/repositories"
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/transports/http"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
)

// Mode represents the application mode in which it runs.
// It can be HTTP, RabbitMQ, or gRPC.
// This is used to determine how the application should behave when started.
// It is defined as a string type for flexibility and clarity.
// The Mode type is used to specify the operational mode of the application.
// It allows the application to adapt its behavior based on the mode it is running in.
// This is particularly useful for applications that can serve multiple purposes or interfaces.
// For example, an application might run as a web server in HTTP mode, consume messages from RabbitMQ in Rabbit mode,
// or provide gRPC services in gRPC mode.
// The Mode type is used to distinguish between these different operational contexts.
// It is defined as a string type for flexibility and clarity.
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

// App creates a new App instance with the provided configuration.
type App struct {
	Cfg    configs.Config
	Logger *zap.Logger
}

// Run initializes the application based on the provided mode and context.
func (a *App) Run(ctx context.Context, mode Mode) error {
	pool, _err := dbs.NewMySQLDB(a.Cfg)
	if condition := _err != nil; condition {
		a.Logger.Error("failed to connect to database", zap.Error(_err))
		return fmt.Errorf("failed to connect to database: %w", _err)
	}
	a.Logger.Info("connected to database successfully")

	// Initialize Example repositories and services
	repo := repositories.NewExampleRepository(pool)
	//add more repositories if needed

	// Create a service register to hold all services
	// This is where the application services are registered.
	// The services are responsible for handling business logic and interacting with repositories.
	serviceRegister := services.Register{
		ExampleService: services.NewExampleService(repo, a.Cfg),
		// add more services to the service register if needed
	}

	switch mode {
	case ModeHTTP:
		h := http.NewHTTPServer(serviceRegister)
		// Start the HTTP server with the provided context and address from the configuration.
		// The server will listen for incoming HTTP requests and handle them using the registered routes.
		a.Logger.Info("starting HTTP server", zap.String("address", a.Cfg.HTTPAddr))
		return h.Run(ctx, a.Cfg.HTTPAddr)
	case ModeRabbit:
		// r := handler.NewPatternRouter()
		// consumer := rabbit.NewResilientConsumer(a.Cfg, r, a.Logger)
		// return consumer.Run(ctx)
		return fmt.Errorf("please setup the %s mode first", mode)
	case ModeGRPC:
		// return grpcx.RunGRPCServer(ctx, svc, a.Cfg.GRPCAddr)
		return fmt.Errorf("please setup the %s mode first", mode)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
}

// New initializes the application with the provided configuration.
// It sets up the logger and prepares the application for running in the specified mode.
// It returns an App instance or an error if initialization fails.
func New(cfg configs.Config) (*App, error) {

	// Prepare Elasticsearch options (could come from cfg)
	esOpts := logs.ESOpts{
		Enabled:       cfg.ElasticEnabled,
		Addresses:     cfg.ElasticAddresses,
		Index:         cfg.ElasticIndex,
		APIKey:        cfg.ElasticAPIKey,
		Username:      cfg.ElasticUsername,
		Password:      cfg.ElasticPassword,
		FlushBytes:    cfg.ElasticBulkFlushBytes,
		FlushInterval: time.Duration(cfg.ElasticBulkFlushIntervalMS) * time.Millisecond,
	}

	// Initialize logger
	logger, stopES, err := logs.NewWithElastic(cfg.AppName, os.Getenv("LOG_TZ"), esOpts)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() {
		stopES()
		_ = logger.Sync()
	}()

	if err != nil {
		return nil, err
	}
	return &App{Cfg: cfg, Logger: logger}, nil
}
