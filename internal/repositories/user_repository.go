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
	DeleteByID(ctx context.Context, id int64) (*entities.User, error)
	GetByUsername(ctx context.Context, username string) (*entities.User, error)
	Create(ctx context.Context, tx *sql.Tx, user *entities.User) (int64, error)

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

func (r *userRepository) DeleteByID(ctx context.Context, id int64) (*entities.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.DeleteByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("user.id", id))

	// 1. Begin the transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	
	// Defer a rollback. If tx.Commit() is called successfully later, 
	// this Rollback does nothing. If the function panics or returns early, it safely aborts.
	defer tx.Rollback()

	// 2. Select the row and lock it using FOR UPDATE
	var user entities.User
	selectQuery := `SELECT id, username, email, created_at, updated_at FROM users WHERE id = ? FOR UPDATE`
	
	err = tx.QueryRowContext(ctx, selectQuery, id).Scan(
		&user.ID, 
		&user.Username, 
		&user.Email, 
		&user.CreatedAt, 
		&user.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Record doesn't exist, nothing to delete
			return nil, nil 
		}
		span.RecordError(err)
		return nil, err
	}

	// 3. Execute the delete operation within the same transaction
	deleteQuery := `DELETE FROM users WHERE id = ?`
	_, err = tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	// 4. Commit the transaction to finalize the deletion
	if err = tx.Commit(); err != nil {
		span.RecordError(err)
		return nil, err
	}

	// 5. Return the user data we fetched in step 2
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

func (r *userRepository) Create(ctx context.Context, tx *sql.Tx, user *entities.User) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Create")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.username", user.Username),
		attribute.String("user.email", user.Email),
	)

	query := `INSERT INTO users (username, email, created_at, updated_at) VALUES (?, ?, ?, ?)`
	now := time.Now()

	result, err := tx.ExecContext(ctx, query, user.Username, user.Email, now, now)
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
