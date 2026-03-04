//go:build wireinject
// +build wireinject

package apps

import (
	"database/sql"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/dbs"
	"go-boilerplate/internal/repositories"
	"go-boilerplate/internal/services"

	// grpchandlers "go-boilerplate/internal/transports/grpc/handlers"
	// grpctransport "go-boilerplate/internal/transports/grpc"
	httptransport "go-boilerplate/internal/transports/http"
	httphandlers "go-boilerplate/internal/transports/http/handlers"
	"go-boilerplate/internal/utils/logs"
	"go-boilerplate/internal/utils/validation"

	"github.com/go-playground/validator/v10"
	"github.com/google/wire"
	"go.uber.org/zap"
)

var InfrastructureProviders = wire.NewSet(
	// Database connection
	dbs.NewMySQLDB,

	// Logger with Elasticsearch support
	logs.ProvideLogger,

	// Input validation
	validation.GetValidator,
)

// Add your repositories here - they handle database operations
var RepositoryProviders = wire.NewSet(
	repositories.NewExampleRepository,
	repositories.NewUserRepository,
	// ......
)

// Add your services here - they contain your business rules
var ServiceProviders = wire.NewSet(
	services.NewExampleService,
	services.NewHealthService,
	services.NewUserService,
	// ....
)

// Add HTTP handlers here - they handle HTTP requests/responses
var HTTPHandlerProviders = wire.NewSet(
	httphandlers.NewExampleHandler,
	httphandlers.NewHealthHandler,
	httphandlers.NewUserHandler,
	// 📝 TO ADD NEW HTTP HANDLER: httphandlers.NewYourHandler,
)

// These start the actual servers (HTTP, gRPC, etc.)
var ServerProviders = wire.NewSet(
	httptransport.NewHTTPServer,
	// grpctransport.NewGRPCServer,
)

// Application represents the complete wired application
type Application struct {
	// Infrastructure
	DB        *sql.DB             // Database connection
	Logger    *zap.Logger         // Logger instance
	Validator *validator.Validate // Input validator

	// Services (Business Logic)
	ExampleService services.ExampleService
	HealthService  services.HealthService
	UserService    services.UserService

	// Transport Servers
	HTTPServer *httptransport.Server
}

// NewApplication creates a new application with all dependencies wired
func NewApplication(
	// Infrastructure
	db *sql.DB,
	logger *zap.Logger,
	validator *validator.Validate,
	// Services
	exampleService services.ExampleService,
	healthService services.HealthService,
	userService services.UserService,
	// Servers
	httpServer *httptransport.Server,
) *Application {
	return &Application{
		DB:             db,
		Logger:         logger,
		Validator:      validator,
		ExampleService: exampleService,
		HealthService:  healthService,
		UserService:    userService,
		HTTPServer:     httpServer,
	}
}

// =============================================================================
// WIRE INJECTORS - These create the actual instances
// =============================================================================

// InitializeApp creates the complete application with all dependencies
func InitializeApp(cfg configs.Config) (*Application, func(), error) {
	wire.Build(
		// Combine all provider sets
		InfrastructureProviders,
		RepositoryProviders,
		ServiceProviders,
		HTTPHandlerProviders,
		// GRPCHandlerProviders,
		ServerProviders,

		// Final assembly
		NewApplication,
	)
	return nil, nil, nil
}

// InitializeHTTPApp creates application optimized for HTTP only
func InitializeHTTPApp(cfg configs.Config) (*httptransport.Server, func(), error) {
	wire.Build(
		InfrastructureProviders,
		RepositoryProviders,
		ServiceProviders,
		HTTPHandlerProviders,
		ServerProviders,
	)
	return nil, nil, nil
}

// InitializeGRPCApp creates application optimized for gRPC only
// func InitializeGRPCApp(cfg configs.Config) (*grpctransport.Server, func(), error) {
// 	wire.Build(
// 		InfrastructureProviders,
// 		RepositoryProviders,
// 		ServiceProviders,
// 		GRPCHandlerProviders,
// 		ServerProviders,
// 	)
// 	return nil, nil, nil
// }

// =============================================================================
// 📖 HOW TO ADD NEW COMPONENTS:
// =============================================================================
//
// 🆕 Adding a new Repository:
// 1. Create: internal/repositories/your_repository.go
// 2. Add: repositories.NewYourRepository to RepositoryProviders
// 3. Run: make wire
//
// 🆕 Adding a new Service:
// 1. Create: internal/services/your_service.go
// 2. Add: services.NewYourService to ServiceProviders
// 3. Add: yourService services.YourService to Application struct
// 4. Add parameter to NewApplication function
// 5. Run: make wire
//
// 🆕 Adding a new HTTP Handler:
// 1. Create: internal/transports/http/handlers/your_handler.go
// 2. Add: httphandlers.NewYourHandler to HTTPHandlerProviders
// 3. Run: make wire
//
// 🆕 Adding a new gRPC Handler:
// 1. Create: internal/transports/grpc/handlers/your_handler.go
// 2. Add: grpchandlers.NewYourHandler to GRPCHandlerProviders
// 3. Run: make wire
//
// That's it! Wire will automatically resolve all dependencies! 🎉
// =============================================================================
