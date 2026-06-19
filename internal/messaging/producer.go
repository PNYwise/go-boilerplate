package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Producer interface {
	PublishJSON(ctx context.Context, exchange, routingKey string, payload interface{}) error
}

type producer struct {
	conn   *RabbitMQConnection
	tracer trace.Tracer
}

// NewProducer creates a new AMQP producer that opens a fresh channel per publish
func NewProducer(conn *RabbitMQConnection) Producer {
	return &producer{
		conn:   conn,
		tracer: otel.Tracer("rabbitmq-producer"),
	}
}

func (p *producer) PublishJSON(ctx context.Context, exchange, routingKey string, payload interface{}) error {
	ctx, span := p.tracer.Start(ctx, fmt.Sprintf("RMQ Producer: %s", routingKey))
	defer span.End()

	if !p.conn.IsConnected() {
		return fmt.Errorf("rabbitmq is not connected")
	}

	channel, err := p.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}
	defer channel.Close()

	// Serialize payload
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Inject OTel trace context into AMQP headers
	headers := make(amqp.Table)
	otel.GetTextMapPropagator().Inject(ctx, headersCarrier(headers))

	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Headers:      headers,
	}

	if err := channel.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}
