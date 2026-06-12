package auditdtos

import "time"

type CreateAuditLogDTO struct {
	Action   string                 `json:"action" validate:"required"`
	Entity   string                 `json:"entity" validate:"required"`
	EntityID string                 `json:"entity_id" validate:"required"`
	Metadata map[string]interface{} `json:"metadata"`
	ActorID  int64                  `json:"actor_id"`
}

type AuditLogMessage struct {
	CreateAuditLogDTO
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"trace_id"`
}
