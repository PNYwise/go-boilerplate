package http

import (
	"context"
	"go-boilerplate/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server holds the Gin engine and services for the HTTP server.
// It is responsible for handling HTTP requests and routing them to the appropriate service methods.
// The Server struct encapsulates the Gin engine and the services it uses, allowing for a clean separation of concerns.
// This structure makes it easier to manage dependencies and maintain the codebase.
type Server struct {
	eng      *gin.Engine
	services services.Register
}

// NewHTTPServer initializes a new HTTP server with the provided services.
// It sets up the Gin engine, applies middleware, and registers routes.
// The server is ready to handle incoming HTTP requests.
// The health check route is also defined here for basic server health monitoring.
func NewHTTPServer(svcs services.Register) *Server {
	r := gin.New()
	r.Use(gin.Recovery())

	// Health route stays here
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Load application routes
	RegisterRoutes(r, svcs)

	return &Server{eng: r, services: svcs}
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
