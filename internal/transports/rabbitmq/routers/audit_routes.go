package routers

import (
	"context"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/transports/rabbitmq/handlers"
)

// RegisterAuditRoutes declares the queue and binds it to the handler
func RegisterAuditRoutes(
	ctx context.Context,
	consumer messaging.Consumer,
	worker *handlers.AuditWorker,
) error {
	queueName := "audit_logs_queue"

	// 1. Declare the queue (idempotent)
	_, err := consumer.DeclareQueue(queueName)
	if err != nil {
		return err
	}

	// 2. Start consuming
	return consumer.Consume(ctx, queueName, worker.HandleAuditLog)
}
