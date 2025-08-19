package entities

import "time"

// ExampleEntity represents an example entity with user, amount, and date fields.
type ExampleEntity struct {
	ID     string    `json:"id"`
	UserID string    `json:"user_id"`
	Amount int64     `json:"amount"`
	Date   time.Time `json:"date"`
}
