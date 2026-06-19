package repositories

import (
	"context"
	"fmt"
	"go-boilerplate/internal/clients/http_clients"
	"go-boilerplate/internal/configs"
	"go-boilerplate/internal/utils/logs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type UserServiceRepository interface {
	GetUserDetailById(ctx context.Context, userId int64) []byte
}

type userServiceRepository struct {
	cfg    configs.Config
	client http_clients.Client
	tracer trace.Tracer
}

func NewUserServiceRepository(client http_clients.Client, cfg configs.Config) UserServiceRepository {
	return &userServiceRepository{
		cfg:    cfg,
		client: client,
		tracer: otel.Tracer("userservice-repository"),
	}
}

func (u *userServiceRepository) GetUserDetailById(ctx context.Context, userId int64) []byte {
	ctx, span := u.tracer.Start(ctx, "UserServiceRepository.GetUserDetailById")
	defer span.End()

	url := fmt.Sprintf("http://localhost:8080/api/v1/users/%d", userId)
	resp, body, err := u.client.Get(ctx, url, nil)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to call user detail API via http client")
	} else {
		logs.SpanInfo(ctx, span, "Successfully hit user detail API via http client",
			attribute.Int("status_code", resp.StatusCode),
			attribute.String("response_body", string(body)),
		)
	}
	// ini harusnya gaboleh ngereturn byte gini sih, harus udah di Unmarshal
	return body
}
