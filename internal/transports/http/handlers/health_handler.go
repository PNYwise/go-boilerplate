package handlers

import (
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/utils/logs"
	"go-boilerplate/internal/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	healthSrv services.HealthService
	tracer    trace.Tracer
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthSrv services.HealthService) *HealthHandler {
	return &HealthHandler{
		healthSrv: healthSrv,
		tracer:    otel.Tracer("health-handler"),
	}
}

// HealthCheck handles GET /health requests
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "HealthHandler.HealthCheck")
	defer span.End()

	if err := h.healthSrv.Check(ctx); err != nil {
		logs.SpanError(ctx, span, err, "Health check failed")
		response.JSON(c, http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := h.healthSrv.GetStatus(ctx)
	logs.SpanInfo(ctx, span, "Health check completed successfully")

	response.JSON(c, http.StatusOK, status)
}

// TestCrash handles GET /test-crash requests - for testing panic recovery
func (h *HealthHandler) TestCrash(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "HealthHandler.TestCrash")
	defer span.End()

	logs.SpanWarn(ctx, span, "Intentional crash test initiated")

	// This will trigger a panic to test our crash recovery
	panic("intentional panic for testing crash recovery")
}
