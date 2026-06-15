package rabbitmq

import (
	"context"
	"go-boilerplate/internal/messaging"
	"go-boilerplate/internal/transports/rabbitmq/handlers"
	"go-boilerplate/internal/transports/rabbitmq/routers"
	"go-boilerplate/internal/utils/logs"

	"go.uber.org/fx"
)

// Server coordinates the initialization and routing for the RabbitMQ consumer
type Server struct {
	consumer    messaging.Consumer
	auditWorker *handlers.AuditWorker
}

func NewRabbitMQServer(
	lc fx.Lifecycle,
	consumer messaging.Consumer,
	auditWorker *handlers.AuditWorker,
) *Server {
	srv := &Server{
		consumer:    consumer,
		auditWorker: auditWorker,
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
	err := routers.RegisterAuditRoutes(ctx, s.consumer, s.auditWorker)
	if err != nil {
		logs.LogError(ctx, err, "Failed to register rabbitmq routes")
		return err
	}
	
	logs.Info("RabbitMQ routes registered and consumer started successfully")
	return nil
}

// Stop cleans up the consumer resources
func (s *Server) Stop() error {
	return s.consumer.Close()
}
