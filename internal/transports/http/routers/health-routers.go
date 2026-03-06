package routers

import (
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/transports/http/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterHealthRoutes(
	r *gin.Engine,
	healthHandler *handlers.HealthHandler,
	cfg configs.Config,
) {
	r.GET("/health", healthHandler.HealthCheck)
	r.GET("/test-crash", healthHandler.TestCrash)
}
