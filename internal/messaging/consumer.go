package messaging

import (
	"context"
	"fmt"
	"go-boilerplate/internal/dbs"
	"go-boilerplate/internal/utils/logs"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/xid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MessageHandler is the signature for a consumer handler function
type MessageHandler func(ctx context.Context, msg amqp.Delivery) error

// Consumer defines the contract for interacting with RabbitMQ as a listener
type Consumer interface {
	DeclareExchange(exchange string, kind string) error
	DeclareQueue(queue string) (amqp.Queue, error)
	BindQueue(queue string, routingKey string, exchange string) error
	Consume(ctx context.Context, queue string, prefetch int, handler MessageHandler) error
	Close() error
}

type consumer struct {
	conn    *dbs.RabbitMQConnection
	tracer  trace.Tracer
}

// NewConsumer creates a consumer wrapper. It doesn't open channels eagerly
// because Fx lifecycle hooks haven't connected to RabbitMQ yet during graph construction.
func NewConsumer(conn *dbs.RabbitMQConnection) (Consumer, error) {
	return &consumer{
		conn:    conn,
		tracer:  otel.Tracer("rabbitmq-consumer"),
	}, nil
}

func (c *consumer) getChannel() (*amqp.Channel, error) {
	if !c.conn.IsConnected() {
		return nil, fmt.Errorf("rabbitmq is not connected")
	}
	return c.conn.Channel()
}

func (c *consumer) DeclareExchange(exchange string, kind string) error {
	ch, err := c.getChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.ExchangeDeclare(
		exchange, // name
		kind,     // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
}

func (c *consumer) DeclareQueue(queue string) (amqp.Queue, error) {
	ch, err := c.getChannel()
	if err != nil {
		return amqp.Queue{}, err
	}
	defer ch.Close()

	return ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
}

func (c *consumer) BindQueue(queue string, routingKey string, exchange string) error {
	ch, err := c.getChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.QueueBind(
		queue,      // queue name
		routingKey, // routing key
		exchange,   // exchange
		false,
		nil,
	)
}

func (c *consumer) Consume(ctx context.Context, queue string, prefetch int, handler MessageHandler) error {
	ch, err := c.getChannel()
	if err != nil {
		return fmt.Errorf("failed to get consumer channel: %w", err)
	}

	// Apply prefetch configuration to this specific channel
	if prefetch > 0 {
		if err := ch.Qos(prefetch, 0, false); err != nil {
			ch.Close()
			return fmt.Errorf("failed to set Qos prefetch: %w", err)
		}
	}

	consumerTag := fmt.Sprintf("%s_consumer_%s", c.conn.AppName(), xid.New().String())

	msgs, err := ch.Consume(
		queue,       // queue
		consumerTag, // consumer name
		false,       // auto-ack (we do manual ack)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		ch.Close()
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	logs.LogInfo(ctx, "RabbitMQ consumer started",
		attribute.String("queue", queue),
		attribute.String("consumer_tag", consumerTag),
		attribute.Int("prefetch", prefetch),
	)

	// Run the listening loop in a goroutine
	go func() {
		defer ch.Close()
		for {
			select {
			case <-ctx.Done():
				return // Context canceled, stop consuming
			case msg, ok := <-msgs:
				if !ok {
					return // Channel closed, stop consuming
				}

				// Create a span for message processing
				processCtx, span := c.tracer.Start(context.Background(), "RabbitMQ.Consume")

				err := handler(processCtx, msg)
				if err != nil {
					logs.SpanError(processCtx, span, err, fmt.Sprintf("Failed to process message from queue: %s", queue))
					// Nack and requeue
					_ = msg.Nack(false, true)
				} else {
					// Ack message
					_ = msg.Ack(false)
				}

				span.End()
			}
		}
	}()

	return nil
}

func (c *consumer) Close() error {
	return nil
}
