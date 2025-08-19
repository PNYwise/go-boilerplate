package exampledtos

import "time"

// ExampleDTO represents the data transfer object for an example entity.
type ExampleDTO struct {
	ID     string    `json:"id"`
	UserID string    `json:"user_id"`
	Amount int64     `json:"amount"`
	Date   time.Time `json:"date"`
}
