package http

import (
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/transports/http/handlers"
	middlewares "go-boilerplate/internal/transports/http/middlewares"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up all API routes grouped by version/module
func RegisterRoutes(
	r *gin.Engine,
	exampleHandler *handlers.ExampleHandler,
	healthHandler *handlers.HealthHandler,
	userHandler *handlers.UserHandler,
	cfg configs.Config,
) {
	// Build middleware from config
	basicAuthMiddleware := middlewares.BasicAuthMiddleware(cfg)

	// Health routes (no auth required)
	r.GET("/health", healthHandler.HealthCheck)
	r.GET("/test-crash", healthHandler.TestCrash) // For testing crash recovery

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Example routes with auth
		exampleGroup := v1.Group("/example", basicAuthMiddleware)
		{
			exampleGroup.POST("/", exampleHandler.CreateExample)
		}

		// User routes with auth
		userGroup := v1.Group("/users", basicAuthMiddleware)
		{
			userGroup.POST("/", userHandler.CreateUser)
			userGroup.GET("/:id", userHandler.GetUserByID)
			userGroup.GET("/username/:username", userHandler.GetUserByUsername)
		}
	}
}
