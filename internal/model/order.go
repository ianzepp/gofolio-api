package model

import "time"

// Order represents a buy/sell/dividend activity.
type Order struct {
	ID              string    `db:"id" json:"id"`
	AccountId       *string   `db:"accountId" json:"accountId,omitempty"`
	AccountUserId   *string   `db:"accountUserId" json:"accountUserId,omitempty"`
	Comment         *string   `db:"comment" json:"comment,omitempty"`
	CreatedAt       time.Time `db:"createdAt" json:"createdAt"`
	Currency        *string   `db:"currency" json:"currency,omitempty"`
	Date            time.Time `db:"date" json:"date"`
	Fee             float64   `db:"fee" json:"fee"`
	IsDraft         bool      `db:"isDraft" json:"isDraft"`
	Quantity        float64   `db:"quantity" json:"quantity"`
	SymbolProfileId string    `db:"symbolProfileId" json:"symbolProfileId"`
	Type            OrderType `db:"type" json:"type"`
	UnitPrice       float64   `db:"unitPrice" json:"unitPrice"`
	UpdatedAt       time.Time `db:"updatedAt" json:"updatedAt"`
	UserId          string    `db:"userId" json:"userId"`
}
