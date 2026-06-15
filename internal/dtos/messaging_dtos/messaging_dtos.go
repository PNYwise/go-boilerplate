package messagingdtos

import (
	"encoding/json"
	"time"
)

// MessageEnvelope is the standard wrapper for all RabbitMQ messages
type MessageEnvelope struct {
	MessageID   string          `json:"message_id"`
	MessageType string          `json:"message_type"` // e.g. "USER_CREATED", "SEND_WELCOME_EMAIL"
	Timestamp   time.Time       `json:"timestamp"`
	TraceID     string          `json:"trace_id"`
	Payload     json.RawMessage `json:"payload"`
}
