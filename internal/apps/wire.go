//go:build wireinject
// +build wireinject

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
// 1. Replace HandlerProviders with gRPC handlers
// 2. Replace TransportProvider with gRPC server
// 3. Update Application struct
//
// DO NOT MODIFY: main.go, app.go
// ===================================================================

import (
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/dbs"
	"go-boilerplate/internal/repositories"
	"go-boilerplate/internal/services"

	httptransport "go-boilerplate/internal/transports/http"
	httphandlers "go-boilerplate/internal/transports/http/handlers"
	dbtransaction "go-boilerplate/internal/utils/db-transaction"
	"go-boilerplate/internal/utils/validation"

	"github.com/google/wire"
)

// ===================================================================
// INFRASTRUCTURE PROVIDERS - Add core dependencies here
// ===================================================================

var InfrastructureProviders = wire.NewSet(
	// Database connection with OpenTelemetry instrumentation
	dbs.NewMySQLDB,
	dbtransaction.NewDbTransactionUtil,
	// Input validation
	validation.GetValidator,

	// Add more infrastructure providers here:
	// redis.NewRedisClient,
	// queue.NewRabbitMQClient,
	// cache.NewCacheProvider,
)

// ===================================================================
// REPOSITORY PROVIDERS - Add your data access layer here
// ===================================================================

var RepositoryProviders = wire.NewSet(
	repositories.NewUserRepository,
	repositories.NewRoleRepository,
	// Add new repositories here:
	// repositories.NewProductRepository,
	// repositories.NewOrderRepository,
)

// ===================================================================
// SERVICE PROVIDERS - Add your business logic layer here
// ===================================================================

var ServiceProviders = wire.NewSet(
	services.NewHealthService,
	services.NewUserService,

	// Add new services here:
	// services.NewProductService,
	// services.NewOrderService,
	// services.NewAuthService,
)

// ===================================================================
// HANDLER PROVIDERS - Current: HTTP (Change this when switching transport)
// ===================================================================

var HandlerProviders = wire.NewSet(
	httphandlers.NewHealthHandler,
	httphandlers.NewUserHandler,

	// Add new handlers here:
	// httphandlers.NewProductHandler,
	// httphandlers.NewAuthHandler,
)

// Future gRPC example:
// var HandlerProviders = wire.NewSet(
//     grpchandlers.NewExampleHandler,
//     grpchandlers.NewUserHandler,
// )

// ===================================================================
// TRANSPORT PROVIDER - Current: HTTP Server (Change this when switching)
// ===================================================================

var TransportProvider = httptransport.NewHTTPServer

// Future gRPC example:
// var TransportProvider = grpctransport.NewGRPCServer

// Future RabbitMQ example:
// var TransportProvider = rmqtransport.NewRabbitMQConsumer
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
// WIRE GENERATION - Single, Simple Builder
// ===================================================================

// InitializeApp creates the application with all dependencies
func InitializeApp(cfg configs.Config) (*Application, func(), error) {
	wire.Build(
		InfrastructureProviders,
		RepositoryProviders,
		ServiceProviders,
		HandlerProviders,
		TransportProvider,
		NewApplication,
	)
	return nil, nil, nil
}
