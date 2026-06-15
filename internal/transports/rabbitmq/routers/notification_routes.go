package routers

import (
	"context"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/transports/rabbitmq/handlers"
)

// RegisterNotificationRoutes declares the queue and binds it to the handler
func RegisterNotificationRoutes(
	ctx context.Context,
	queueName string,
	prefetch int,
	consumer messaging.Consumer,
	worker *handlers.NotificationWorker,
) error {

	// 1. Declare the queue
	_, err := consumer.DeclareQueue(queueName)
	if err != nil {
		return err
	}

	// 2. Start consuming
	return consumer.Consume(ctx, queueName, prefetch, worker.HandleNotification)
}
