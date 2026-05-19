package repositories

import (
	"context"
	"database/sql"
	"time"

	"go-boilerplate/internal/entities"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// RoleRepository defines operations for role data access
type RoleRepository interface {
	Create(ctx context.Context, tx *sql.Tx, role *entities.Role) (int64, error)
}

type roleRepository struct {
	db     *sql.DB
	tracer trace.Tracer
}

// NewRoleRepository creates a new Role repository instance
func NewRoleRepository(db *sql.DB) RoleRepository {
	return &roleRepository{
		db:     db,
		tracer: otel.Tracer("role-repository"),
	}
}

// Create inserts a new role into the database
func (r *roleRepository) Create(ctx context.Context, tx *sql.Tx, role *entities.Role) (int64, error) {
	ctx, span := r.tracer.Start(ctx, "RoleRepository.Create")
	defer span.End()

	now := time.Now()

	query := "INSERT INTO roles (name, created_at, updated_at) VALUES (?, ?, ?)"
	result, err := tx.ExecContext(ctx, query, role.Name, now, now)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}
