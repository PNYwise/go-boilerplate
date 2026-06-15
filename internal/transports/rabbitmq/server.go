package rabbitmq

import (
	"context"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/transports/rabbitmq/handlers"
	"go-boilerplate/internal/transports/rabbitmq/routers"
	"go-boilerplate/internal/utils/logs"

	"go.uber.org/fx"
)

// Server coordinates the initialization and routing for the RabbitMQ consumer
type Server struct {
	config             configs.Config
	consumer           messaging.Consumer
	auditWorker        *handlers.AuditWorker
	notificationWorker *handlers.NotificationWorker
}

func NewRabbitMQServer(
	lc fx.Lifecycle,
	config configs.Config,
	consumer messaging.Consumer,
	auditWorker *handlers.AuditWorker,
	notificationWorker *handlers.NotificationWorker,
) *Server {
	srv := &Server{
		config:             config,
		consumer:           consumer,
		auditWorker:        auditWorker,
		notificationWorker: notificationWorker,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return srv.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return srv.Stop()
		},
	})

	return srv
}

// Start begins consuming messages
func (s *Server) Start(ctx context.Context) error {
	// Register all rabbitmq consumer routes
	err := routers.RegisterAuditRoutes(ctx, s.config.RabbitAuditQueue, s.config.RabbitAuditPrefetch, s.consumer, s.auditWorker)
	if err != nil {
		logs.LogError(ctx, err, "Failed to register rabbitmq audit routes")
		return err
	}

	err = routers.RegisterNotificationRoutes(ctx, s.config.RabbitNotificationQueue, s.config.RabbitNotificationPrefetch, s.consumer, s.notificationWorker)
	if err != nil {
		logs.LogError(ctx, err, "Failed to register rabbitmq notification routes")
		return err
	}
	
	logs.Info("RabbitMQ routes registered and consumer started successfully")
	return nil
}

// Stop cleans up the consumer resources
func (s *Server) Stop() error {
	return s.consumer.Close()
}
