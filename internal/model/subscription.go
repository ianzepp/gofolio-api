package model

import "time"

// Subscription tracks a user's premium subscription period.
type Subscription struct {
	ID        string    `db:"id" json:"id"`
	CreatedAt time.Time `db:"createdAt" json:"createdAt"`
	ExpiresAt time.Time `db:"expiresAt" json:"expiresAt"`
	Price     *float64  `db:"price" json:"price,omitempty"`
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
	UserId    string    `db:"userId" json:"userId"`
}
