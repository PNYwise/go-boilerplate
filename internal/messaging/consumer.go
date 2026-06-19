package messaging

import (
	"context"
	"fmt"
	"go-boilerplate/internal/utils/logs"
	"time"

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
	DeclareQueue(queue string, args amqp.Table) (amqp.Queue, error)
	BindQueue(queue string, routingKey string, exchange string) error
	Consume(ctx context.Context, queue string, prefetch int, handler MessageHandler) error
	Close() error
}

type consumer struct {
	conn   *RabbitMQConnection
	tracer trace.Tracer
}

// NewConsumer creates a consumer wrapper. It doesn't open channels eagerly
// because Fx lifecycle hooks haven't connected to RabbitMQ yet during graph construction.
func NewConsumer(conn *RabbitMQConnection) (Consumer, error) {
	return &consumer{
		conn:   conn,
		tracer: otel.Tracer("rabbitmq-consumer"),
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

func (c *consumer) DeclareQueue(queue string, args amqp.Table) (amqp.Queue, error) {
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
		args,  // arguments
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
		defer func() {
			if ch != nil {
				ch.Close()
			}
		}()

		for {
		ProcessLoop:
			for {
				select {
				case <-ctx.Done():
					return // Context canceled, stop consuming
				case msg, ok := <-msgs:
					if !ok {
						break ProcessLoop // Channel closed, stop inner loop to reconnect
					}

					// Extract trace context from AMQP headers
					ctxWithTrace := otel.GetTextMapPropagator().Extract(ctx, headersCarrier(msg.Headers))

					// Create a span for message processing using the inherited context
					processCtx, span := c.tracer.Start(ctxWithTrace, fmt.Sprintf("RMQ Consumer: %s", queue))

					err := handler(processCtx, msg)
					if err != nil {
						logs.SpanError(processCtx, span, err, fmt.Sprintf("Failed to process message from queue: %s", queue))
						// Nack and discard (this routes it to the DLX automatically)
						if nackErr := msg.Nack(false, false); nackErr != nil {
							logs.SpanError(processCtx, span, nackErr, "Failed to Nack message")
						}
					} else {
						// Ack message
						if ackErr := msg.Ack(false); ackErr != nil {
							logs.SpanError(processCtx, span, ackErr, "Failed to Ack message")
						}
					}

					span.End()
				}
			}

			if ctx.Err() != nil {
				return // Context canceled during ProcessLoop
			}

			// If we got here, the channel was closed. Reconnect loop:
			logs.LogWarn(ctx, "RabbitMQ consumer channel closed. Attempting to reconnect...", attribute.String("queue", queue))

			ticker := time.NewTicker(3 * time.Second)
		ReconnectLoop:
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					// Wait for the underlying connection to be restored by rabbitmq.go
					if !c.conn.IsConnected() {
						continue // Still disconnected, wait and try again
					}

					newCh, err := c.getChannel()
					if err != nil {
						logs.LogError(ctx, err, "Failed to get new channel during reconnect")
						continue
					}

					if prefetch > 0 {
						if err := newCh.Qos(prefetch, 0, false); err != nil {
							newCh.Close()
							continue
						}
					}

					newMsgs, err := newCh.Consume(queue, consumerTag, false, false, false, false, nil)
					if err != nil {
						newCh.Close()
						logs.LogError(ctx, err, "Failed to resume consuming during reconnect")
						continue
					}

					ch = newCh
					msgs = newMsgs
					logs.LogInfo(ctx, "RabbitMQ consumer reconnected successfully", attribute.String("queue", queue))
					ticker.Stop()
					break ReconnectLoop
				}
			}
		}
	}()

	return nil
}

func (c *consumer) Close() error {
	return nil
}
