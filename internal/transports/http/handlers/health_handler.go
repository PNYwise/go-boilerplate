package handlers

import (
	"go-boilerplate/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
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
	if err := h.healthSrv.Check(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	status := h.healthSrv.GetStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}
