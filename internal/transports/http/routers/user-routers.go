package routers

import (
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/transports/http/handlers"
	"go-boilerplate/internal/transports/http/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(
	r *gin.Engine,
	userHandler *handlers.UserHandler,
	cfg configs.Config,
) {
	// Group routes with common middleware
	api := r.Group("/api/v1")

	// Protected user routes with basic auth
	protectedUserApi := api.Group("/users")
	protectedUserApi.Use(middlewares.BasicAuthMiddleware(cfg))
	{
		protectedUserApi.POST("with-role", userHandler.CreateUserWithRole)
		protectedUserApi.POST("/", userHandler.CreateUser)
		protectedUserApi.GET("/:id", userHandler.GetUserByID)
		protectedUserApi.GET("/username/:username", userHandler.GetUserByUsername)
	}
}
