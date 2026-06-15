package services

import (
	"context"
	"fmt"
	"go-boilerplate/internal/utils/logs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type NotificationService interface {
	SendWelcomeEmail(ctx context.Context, email string, name string) error
	SendSMS(ctx context.Context, phoneNumber string, message string) error
}

type notificationService struct {
	tracer trace.Tracer
}

func NewNotificationService() NotificationService {
	return &notificationService{
		tracer: otel.Tracer("notification-service"),
	}
}

func (s *notificationService) SendWelcomeEmail(ctx context.Context, email string, name string) error {
	ctx, span := s.tracer.Start(ctx, "NotificationService.SendWelcomeEmail")
	defer span.End()

	logs.SpanInfo(ctx, span, fmt.Sprintf("SIMULATION: Sending Welcome Email to %s (%s)", name, email))
	// Actual email logic goes here
	return nil
}

func (s *notificationService) SendSMS(ctx context.Context, phoneNumber string, message string) error {
	ctx, span := s.tracer.Start(ctx, "NotificationService.SendSMS")
	defer span.End()

	logs.SpanInfo(ctx, span, fmt.Sprintf("SIMULATION: Sending SMS to %s: %s", phoneNumber, message))
	// Actual SMS logic goes here
	return nil
}
