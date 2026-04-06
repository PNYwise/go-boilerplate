package http

import (
	"context"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/transports/http/handlers"
	"go-boilerplate/internal/transports/http/routers"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Server holds the Gin engine and services for the HTTP server.
// It is responsible for handling HTTP requests and routing them to the appropriate service methods.
// The Server struct encapsulates the Gin engine and the services it uses, allowing for a clean separation of concerns.
// This structure makes it easier to manage dependencies and maintain the codebase.
type Server struct {
	eng *gin.Engine
}

// NewHTTPServer initializes a new HTTP server with OpenTelemetry instrumentation.
// It sets up the Gin engine with automatic request tracing, applies middleware, and registers routes.
// All HTTP requests are automatically traced and metrics are collected for observability.
func NewHTTPServer(
	healthHandler *handlers.HealthHandler,
	userHandler *handlers.UserHandler,
	cfg configs.Config,
) *Server {
	r := gin.New()

	// Add OpenTelemetry middleware for automatic request tracing
	r.Use(otelgin.Middleware(cfg.AppName))

	// Standard middleware
	r.Use(gin.Recovery())

	// Health route stays here
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Load application routes
	routers.RegisterUserRoutes(r, userHandler, cfg)
	routers.RegisterHealthRoutes(r, healthHandler, cfg)

	return &Server{eng: r}
}

// Run starts the HTTP server and listens for incoming requests.
// It blocks until the context is done, allowing for graceful shutdown.
// The server listens on the specified address and handles requests using the Gin engine.
// If the context is canceled, it gracefully shuts down the server.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.eng,
	}
	go func() {
		_ = srv.ListenAndServe()
	}()
	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shCtx)
}
