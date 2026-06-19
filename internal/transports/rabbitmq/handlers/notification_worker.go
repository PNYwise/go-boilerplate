package handlers

import (
	"context"
	"encoding/json"
	messagingdtos "go-boilerplate/internal/dtos/messaging_dtos"
	"go-boilerplate/internal/services"
	"go-boilerplate/internal/utils/logs"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type NotificationWorker struct {
	notificationSrv services.NotificationService
	tracer          trace.Tracer
}

func NewNotificationWorker(notificationSrv services.NotificationService) *NotificationWorker {
	return &NotificationWorker{
		notificationSrv: notificationSrv,
		tracer:          otel.Tracer("notification-worker"),
	}
}

// HandleNotification processes messages from the notification queue
func (h *NotificationWorker) HandleNotification(ctx context.Context, msg amqp.Delivery) error {
	ctx, span := h.tracer.Start(ctx, "NotificationWorker.HandleNotification")
	defer span.End()

	var envelope messagingdtos.MessageEnvelope
	if err := json.Unmarshal(msg.Body, &envelope); err != nil {
		logs.SpanError(ctx, span, err, "Failed to unmarshal message envelope")
		return err // Returning error causes Nack
	}

	logs.SpanInfo(ctx, span, "Received notification message: "+envelope.MessageType)

	// Switch on the MessageType to handle multiple commands in the same queue
	switch envelope.MessageType {
	case "USER_WELCOME_EMAIL":
		var payload struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			logs.SpanError(ctx, span, err, "Failed to unmarshal USER_WELCOME_EMAIL payload")
			return err
		}
		if err := h.notificationSrv.SendWelcomeEmail(ctx, payload.Email, payload.Name); err != nil {
			logs.SpanError(ctx, span, err, "Failed to send welcome email")
			return err
		}

	case "PASSWORD_RESET_SMS":
		var payload struct {
			PhoneNumber string `json:"phone_number"`
			Message     string `json:"message"`
		}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			logs.SpanError(ctx, span, err, "Failed to unmarshal PASSWORD_RESET_SMS payload")
			return err
		}
		if err := h.notificationSrv.SendSMS(ctx, payload.PhoneNumber, payload.Message); err != nil {
			logs.SpanError(ctx, span, err, "Failed to send SMS")
			return err
		}

	default:
		// Unknown message type, we can either ignore and ack, or return an error to nack.
		// Usually we log a warning and ack it so it doesn't block the queue.
		logs.SpanInfo(ctx, span, "Unknown message type: "+envelope.MessageType)
	}

	return nil
}
