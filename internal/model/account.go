package model

import "time"

// Account represents a user's financial account.
// Composite PK: (id, userId)
type Account struct {
	ID         string    `db:"id" json:"id"`
	UserId     string    `db:"userId" json:"userId"`
	Balance    float64   `db:"balance" json:"balance"`
	Comment    *string   `db:"comment" json:"comment,omitempty"`
	CreatedAt  time.Time `db:"createdAt" json:"createdAt"`
	Currency   *string   `db:"currency" json:"currency,omitempty"`
	IsExcluded bool      `db:"isExcluded" json:"isExcluded"`
	Name       *string   `db:"name" json:"name,omitempty"`
	PlatformId *string   `db:"platformId" json:"platformId,omitempty"`
	UpdatedAt  time.Time `db:"updatedAt" json:"updatedAt"`
}

// AccountBalance records a point-in-time balance for an account.
// Unique: (accountId, date)
type AccountBalance struct {
	ID        string    `db:"id" json:"id"`
	AccountId string    `db:"accountId" json:"accountId"`
	CreatedAt time.Time `db:"createdAt" json:"createdAt"`
	Date      time.Time `db:"date" json:"date"`
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
	UserId    string    `db:"userId" json:"userId"`
	Value     float64   `db:"value" json:"value"`
}
