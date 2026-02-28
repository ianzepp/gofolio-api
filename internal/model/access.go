package model

import (
	"encoding/json"
	"time"
)

// Access grants one user access to another user's portfolio.
type Access struct {
	ID            string           `db:"id" json:"id"`
	Alias         *string          `db:"alias" json:"alias,omitempty"`
	CreatedAt     time.Time        `db:"createdAt" json:"createdAt"`
	GranteeUserId *string          `db:"granteeUserId" json:"granteeUserId,omitempty"`
	Permissions   []string          `db:"permissions" json:"permissions"` // Postgres enum array
	Settings      json.RawMessage  `db:"settings" json:"settings"`
	UpdatedAt     time.Time        `db:"updatedAt" json:"updatedAt"`
	UserId        string           `db:"userId" json:"userId"`
}
