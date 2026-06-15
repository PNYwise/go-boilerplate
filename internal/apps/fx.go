package apps

// ===================================================================
// DEVELOPER CUSTOMIZATION ZONE - MODIFY THIS FILE TO ADD FEATURES
// ===================================================================
//
// This file is where you add:
// - New repositories
// - New services
// - New handlers
// - New providers
//
// When you want to switch to gRPC later:
// 1. Replace HandlerModule with gRPC handlers
// 2. Replace TransportModule with gRPC server
// 3. Update Application struct
//
// DO NOT MODIFY: main.go, app.go
// ===================================================================

import (
	"context"
	"database/sql"
	"fmt"

	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/dbs"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/repositories"
	"go-boilerplate/internal/services"

	httptransport "go-boilerplate/internal/transports/http"
	httphandlers "go-boilerplate/internal/transports/http/handlers"
	"go-boilerplate/internal/transports/rabbitmq"
	rabbitmqhandlers "go-boilerplate/internal/transports/rabbitmq/handlers"
	dbtransaction "go-boilerplate/internal/utils/db-transaction"
	"go-boilerplate/internal/utils/validation"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// ===================================================================
// APPLICATION STRUCTURE
// ===================================================================

// Application holds the main application server
type Application struct {
	Server *httptransport.Server // Change type when switching transport
}

// NewApplication creates a new Application
func NewApplication(
	server *httptransport.Server, // Change type when switching transport
) *Application {
	return &Application{
		Server: server,
	}
}

// ===================================================================
// INFRASTRUCTURE MODULE - Add core dependencies here
// ===================================================================

// newManagedMySQLDB wraps dbs.NewMySQLDB to register the cleanup function
// with the FX lifecycle so the connection is closed gracefully on app stop.
func newManagedMySQLDB(lc fx.Lifecycle, cfg configs.Config) (*sql.DB, error) {
	db, cleanup, err := dbs.NewMySQLDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("newManagedMySQLDB: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cleanup()
			return nil
		},
	})

	return db, nil
}

// newManagedRedisClient wraps dbs.NewRedisClient to register the cleanup function
// with the FX lifecycle so the connection is closed gracefully on app stop.
func newManagedRedisClient(lc fx.Lifecycle, cfg configs.Config) (*redis.Client, error) {
	client, err := dbs.NewRedisClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("newManagedRedisClient: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			return client.Close()
		},
	})

	return client, nil
}

var InfrastructureModule = fx.Module("infrastructure",
	fx.Provide(
		// Database: cleanup registered via fx.Lifecycle in newManagedMySQLDB
		newManagedMySQLDB,
		newManagedRedisClient,
		dbs.NewRabbitMQConnection,
		dbs.NewRedsync,
		dbtransaction.NewDbTransactionUtil,
		messaging.NewProducer,
		messaging.NewConsumer,
		// Input validation (singleton via sync.Once internally)
		validation.GetValidator,

		// Add more infrastructure providers here:
		// redis.NewRedisClient,
		// queue.NewRabbitMQClient,
		// cache.NewCacheProvider,
	),
	fx.Invoke(func(r *redis.Client, q *dbs.RabbitMQConnection) {
		// Eagerly initialize Redis and RabbitMQ connections on startup
		// to ensure the app fails fast if infrastructure is unreachable.
	}),
)

// ===================================================================
// REPOSITORY MODULE - Add your data access layer here
// ===================================================================

var RepositoryModule = fx.Module("repositories",
	fx.Provide(
		repositories.NewUserRepository,
		repositories.NewRoleRepository,
		// Add new repositories here:
		// repositories.NewProductRepository,
		// repositories.NewOrderRepository,
	),
)

// ===================================================================
// SERVICE MODULE - Add your business logic layer here
// ===================================================================

var ServiceModule = fx.Module("services",
	fx.Provide(
		services.NewHealthService,
		services.NewUserService,
		services.NewAuditService,

		// Add new services here:
		// services.NewProductService,
		// services.NewOrderService,
		// services.NewAuthService,
	),
)

// ===================================================================
// HANDLER MODULE - Current: HTTP (Change this when switching transport)
// ===================================================================

var HandlerModule = fx.Module("handlers",
	fx.Provide(
		httphandlers.NewHealthHandler,
		httphandlers.NewUserHandler,
		httphandlers.NewAuditHandler,
		rabbitmqhandlers.NewAuditWorker,

		// Add new handlers here:
		// httphandlers.NewProductHandler,
		// httphandlers.NewAuthHandler,
	),
)

// Future gRPC example:
// var HandlerModule = fx.Module("handlers", fx.Provide(
//     grpchandlers.NewExampleHandler,
//     grpchandlers.NewUserHandler,
// ))

// ===================================================================
// TRANSPORT MODULE - Current: HTTP Server (Change this when switching)
// ===================================================================

var TransportModule = fx.Module("transport",
	fx.Provide(
		httptransport.NewHTTPServer,
		rabbitmq.NewRabbitMQServer,
	),
	fx.Invoke(func(*rabbitmq.Server) {
		// Eagerly instantiate RabbitMQServer to ensure its lifecycle hooks (Start/Stop) run
	}),
)

// Future gRPC example:
// var TransportModule = fx.Module("transport", fx.Provide(grpctransport.NewGRPCServer))

// Future RabbitMQ example:
// var TransportModule = fx.Module("transport", fx.Provide(rmqtransport.NewRabbitMQConsumer))

// ===================================================================
// FX APP BUILDER
// ===================================================================

// InitializeApp creates the Application and all its dependencies using Uber FX.
// Returns the Application and a cleanup function to stop the FX container,
// mirroring the signature previously generated by Wire.
func InitializeApp(cfg configs.Config) (*Application, func(), error) {
	var app *Application

	fxApp := fx.New(
		// Suppress FX's default startup/shutdown printing
		fx.NopLogger,

		// Make the typed config value available to all providers
		fx.Supply(cfg),

		InfrastructureModule,
		RepositoryModule,
		ServiceModule,
		HandlerModule,
		TransportModule,

		fx.Provide(NewApplication),

		// Populate our local pointer so we can return it
		fx.Populate(&app),
	)

	if err := fxApp.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to start fx container: %w", err)
	}

	cleanup := func() {
		_ = fxApp.Stop(context.Background())
	}

	return app, cleanup, nil
}
