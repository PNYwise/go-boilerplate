# Go Boilerplate - Simplified Architecture

## 🎯 Ultra-Simple, Single Transport Architecture

This boilerplate has been simplified to focus on **single transport mode** (HTTP by default) with easy switching to gRPC or other transports when needed. No more complex mode switching - just pure, clean code.

## 📁 File Structure & Ownership

### 🚫 **DO NOT MODIFY** (Infrastructure Code)
- `cmd/main.go` - Ultra-minimal application bootstrap
- `internal/apps/app.go` - Core application lifecycle management

### ✅ **DEVELOPER ZONE** (Modify These)
- **`internal/apps/wire.go`** - **PRIMARY CUSTOMIZATION FILE** 🎯
- `internal/repositories/` - Data access layer
- `internal/services/` - Business logic layer  
- `internal/transports/http/handlers/` - HTTP request handlers
- `internal/dtos/` - Data transfer objects
- `internal/entities/` - Domain entities
- `internal/configs/config.go` - Configuration

## 🚀 Development Workflow

### Adding New Features (All in wire.go)

```go
// 1. Add repository
var RepositoryProviders = wire.NewSet(
    repositories.NewExampleRepository,
    repositories.NewUserRepository,
    repositories.NewProductRepository, // ← Add here
)

// 2. Add service  
var ServiceProviders = wire.NewSet(
    services.NewExampleService,
    services.NewUserService,
    services.NewProductService, // ← Add here
)

// 3. Add handler
var HandlerProviders = wire.NewSet(
    httphandlers.NewExampleHandler,
    httphandlers.NewUserHandler,
    httphandlers.NewProductHandler, // ← Add here
)
```

### Switching to gRPC (Simple 3-Step Process)

```go
// Step 1: Replace handlers
var HandlerProviders = wire.NewSet(
    grpchandlers.NewExampleHandler,  // HTTP → gRPC
    grpchandlers.NewUserHandler,
)

// Step 2: Replace transport
var TransportProvider = grpctransport.NewGRPCServer  // HTTP → gRPC

// Step 3: Update Application struct
type Application struct {
    Server *grpctransport.Server  // Change type
    Logger *zap.Logger
}

func NewApplication(server *grpctransport.Server, logger *zap.Logger) *Application {
    return &Application{Server: server, Logger: logger}
}
```

That's it! No mode switching, no complex configuration.

## 🏃‍♂️ Running the Application

```bash
# Simple - no modes to worry about
go run ./cmd

# With staging config
go run ./cmd -stage=staging

# Production
go run ./cmd -stage=prod
```

## ⚡ Key Simplifications

### Before (Complex)
- Mode system with HTTP/gRPC/Rabbit switching
- Multiple builder functions
- Complex app_builder.go file
- Mode parsing logic in main.go

### After (Simple)  
- Single transport focus (HTTP → gRPC when ready)
- One builder: `InitializeApp`
- Ultra-minimal main.go (no mode logic)
- Pure wire.go customization

## 🧪 Example: Adding Product Feature

**1. Create files:**
```bash
internal/repositories/product_repository.go
internal/services/product_service.go  
internal/transports/http/handlers/product_handler.go
```

**2. Update wire.go (only file you need to touch):**
```go
var RepositoryProviders = wire.NewSet(
    // existing...
    repositories.NewProductRepository,
)

var ServiceProviders = wire.NewSet(
    // existing...  
    services.NewProductService,
)

var HandlerProviders = wire.NewSet(
    // existing...
    httphandlers.NewProductHandler,
)
```

**3. Add routes in routers.go:**
```go
productGroup := v1.Group("/products")
{
    productGroup.GET("/", productHandler.GetProducts)
    productGroup.POST("/", productHandler.CreateProduct)
}
```

Done! Wire automatically handles all dependency injection.

## 🛡️ Built-in Safety

- **Panic recovery** at all levels
- **Resource cleanup** automatically chained
- **Graceful shutdown** on signals
- **Zero resource leaks** guaranteed

## 🎯 Philosophy

**"One thing, done perfectly"** - Focus on your current transport (HTTP), add others when needed. No premature complexity, just clean, maintainable code that scales.