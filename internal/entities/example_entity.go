package entities

import "time"

// ExampleEntity represents an example entity with user, amount, and date fields.
type ExampleEntity struct {
	ID     string    `validate:"required"`
	UserID string    `validate:"required"`
	Amount int64     `validate:"required,gt=0"`
	Date   time.Time `validate:"required"`
}
