package repositories

import (
	"context"
	"database/sql"
	"go-boilerplate/internal/entities"
	"time"
)

// ExampleRepository defines the interface for interacting with the ExampleEntity in the database.
// It provides methods for retrieving and creating ExampleEntity records.
// This interface abstracts the database operations, allowing for easier testing and flexibility in implementation.
// The ExampleRepository interface is designed to encapsulate the data access logic for ExampleEntity.
// It provides methods to interact with the underlying database, allowing for operations such as retrieving an entity by ID or creating a new entity.
// This abstraction enables easier testing and flexibility in implementation, as different database backends can be used without changing the service layer.
type ExampleRepository interface {
	GetByID(ctx context.Context, id int64) (*entities.ExampleEntity, error)
	Create(ctx context.Context, u *entities.ExampleEntity) (int64, error)
}

type exampleRepository struct {
	db *sql.DB
}
// NewExampleRepository creates a new instance of ExampleRepository using the provided database connection.
// It is responsible for interacting with the database to perform CRUD operations on ExampleEntity.
func NewExampleRepository(db *sql.DB) ExampleRepository {
	return &exampleRepository{db: db}
}

func (r *exampleRepository) GetByID(ctx context.Context, id int64) (*entities.ExampleEntity, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, amount FROM users WHERE id = $1`, id)
	var u entities.ExampleEntity
	if err := row.Scan(&u.ID, &u.UserID, &u.Amount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *exampleRepository) Create(ctx context.Context, u *entities.ExampleEntity) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (user_id, amount) VALUES ($1, $2) RETURNING id`,
		u.UserID, u.Amount,
	).Scan(&id)
	return id, err
}
