package handlers

import (
	auditdtos "go-boilerplate/internal/dtos/audit_dtos"
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/utils/logs"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type AuditHandler struct {
	auditSrv services.AuditService
	tracer   trace.Tracer
}

// NewAuditHandler creates a new AuditHandler
func NewAuditHandler(auditSrv services.AuditService) *AuditHandler {
	return &AuditHandler{
		auditSrv: auditSrv,
		tracer:   otel.Tracer("audit-handler"),
	}
}

// CreateAuditLog handles POST /audit-logs
func (h *AuditHandler) CreateAuditLog(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "AuditHandler.CreateAuditLog")
	defer span.End()

	var req auditdtos.CreateAuditLogDTO
	if err := c.BindJSON(&req); err != nil {
		logs.SpanError(ctx, span, err, "Failed to bind JSON request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.auditSrv.PublishAuditLog(ctx, req); err != nil {
		logs.SpanError(ctx, span, err, "Failed to publish audit log")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Audit log accepted for processing"})
}
