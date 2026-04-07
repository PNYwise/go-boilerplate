package repositories

import (
	"context"
	"database/sql"
	"go-boilerplate/internal/entities"
	"go-boilerplate/internal/utils/logs"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserRepository defines operations for user data access
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	Create(ctx context.Context, user *entities.User) (int64, error)
}

type userRepository struct {
	db     *sql.DB
	tracer trace.Tracer
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{
		db:     db,
		tracer: otel.Tracer("user-repository"),
	}
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*entities.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", id))

	query := `SELECT id, username, email, created_at, updated_at FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var user entities.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		span.RecordError(err)
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*entities.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetByUsername")
	defer span.End()

	span.SetAttributes(attribute.String("user.username", username))

	query := `SELECT id, username, email, created_at, updated_at FROM users WHERE username = ?`
	row := r.db.QueryRowContext(ctx, query, username)

	var user entities.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		span.RecordError(err)
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) Create(ctx context.Context, user *entities.User) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.username", user.Username),
		attribute.String("user.email", user.Email),
	)

	query := `INSERT INTO users (username, email, created_at, updated_at) VALUES (?, ?, ?, ?)`
	now := time.Now()

	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, now, now)
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to insert user into database",
			attribute.String("username", user.Username),
		)
		return 0, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		logs.SpanError(ctx, span, err, "Failed to get last insert ID",
			attribute.String("username", user.Username),
		)
		return 0, err
	}

	span.SetAttributes(attribute.Int64("user.id", id))
	return id, nil
}
