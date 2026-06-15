package routers

import (
	"context"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/transports/rabbitmq/handlers"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RegisterAuditRoutes declares the queue and binds it to the handler
func RegisterAuditRoutes(
	ctx context.Context,
	dlxExchange string,
	queueName string,
	prefetch int,
	consumer messaging.Consumer,
	worker *handlers.AuditWorker,
) error {

	// 1. Declare the queue with DLX arguments
	args := amqp.Table{}
	if dlxExchange != "" {
		args["x-dead-letter-exchange"] = dlxExchange
	}

	_, err := consumer.DeclareQueue(queueName, args)
	if err != nil {
		return err
	}

	// 2. Start consuming
	return consumer.Consume(ctx, queueName, prefetch, worker.HandleAuditLog)
}
