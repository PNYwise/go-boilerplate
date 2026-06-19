package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	messagingdtos "go-boilerplate/internal/dtos/messaging_dtos"
	"go-boilerplate/internal/messaging"
	"time"

	"github.com/rs/xid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type PublishMessageRepository interface {
	PublishMessage(ctx context.Context, exchange, routingKey, messageType string, payload interface{}) error
}

type rmqPublishMessageRepository struct {
	producer messaging.Producer
	tracer   trace.Tracer
}

// NewRoleRepository creates a new Role repository instance
func NewRmqPublishMessageRepository(producer messaging.Producer) PublishMessageRepository {
	return &rmqPublishMessageRepository{
		producer: producer,
		tracer:   otel.Tracer("role-repository"),
	}
}

func (p *rmqPublishMessageRepository) PublishMessage(ctx context.Context, exchange, routingKey, messageType string, payload interface{}) error {
	ctx, span := p.tracer.Start(ctx, "PublishMessageRepository.PublishMessage")
	defer span.End()

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal inner payload: %w", err)
	}

	msgID := xid.New().String()

	envelope := messagingdtos.MessageEnvelope{
		MessageID:   msgID,
		MessageType: messageType,
		Timestamp:   time.Now().UTC(),
		TraceID:     span.SpanContext().TraceID().String(),
		Payload:     rawPayload,
	}

	return p.producer.PublishJSON(ctx, exchange, routingKey, envelope)
}
