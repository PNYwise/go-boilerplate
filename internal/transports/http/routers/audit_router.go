package routers

import (
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/transports/http/handlers"

	"github.com/gin-gonic/gin"
)

// RegisterAuditRoutes registers the HTTP routes for the audit logs
func RegisterAuditRoutes(router *gin.Engine, handler *handlers.AuditHandler, cfg configs.Config) {
	// Group routes under /api/v1/audit-logs
	auditGroup := router.Group("/api/v1/audit-logs")
	{
		auditGroup.POST("/", handler.CreateAuditLog)
	}
}
