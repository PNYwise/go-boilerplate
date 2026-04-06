package handlers

import (
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/utils/logs"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	healthSrv services.HealthService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthSrv services.HealthService) *HealthHandler {
	return &HealthHandler{
		healthSrv: healthSrv,
	}
}

// HealthCheck handles GET /health requests
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	logs.LogInfo(ctx, "Health check request received")

	if err := h.healthSrv.Check(ctx); err != nil {
		logs.LogError(ctx, err, "Health check failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	status := h.healthSrv.GetStatus(ctx)

	logs.LogInfo(ctx, "Health check completed successfully",
		attribute.String("status", "ok"),
	)

	c.JSON(http.StatusOK, status)
}

// TestCrash handles GET /test-crash requests - for testing panic recovery
func (h *HealthHandler) TestCrash(c *gin.Context) {
	// This will trigger a panic to test our crash recovery
	panic("intentional panic for testing crash recovery")
}
