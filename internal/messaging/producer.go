package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/dbs"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Producer interface {
	PublishJSON(ctx context.Context, exchange, routingKey string, payload interface{}) error
}

type producer struct {
	conn   *dbs.RabbitMQConnection
	tracer trace.Tracer
}

// NewProducer creates a new AMQP producer that opens a fresh channel per publish
func NewProducer(conn *dbs.RabbitMQConnection) Producer {
	return &producer{
		conn:   conn,
		tracer: otel.Tracer("rabbitmq-producer"),
	}
}

func (p *producer) PublishJSON(ctx context.Context, exchange, routingKey string, payload interface{}) error {
	ctx, span := p.tracer.Start(ctx, "RabbitMQ.Publish")
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

// headersCarrier adapts amqp.Table to satisfy the propagation.TextMapCarrier interface
type headersCarrier amqp.Table

func (c headersCarrier) Get(key string) string {
	val, ok := c[key]
	if !ok {
		return ""
	}
	strVal, ok := val.(string)
	if !ok {
		return ""
	}
	return strVal
}

func (c headersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c headersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
