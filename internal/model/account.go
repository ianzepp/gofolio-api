package model

import "time"

// Account represents a brokerage, bank, or cash account that a user tracks in
// Ghostfolio. Orders (transactions) are associated with an Account to enable
// per-account performance reporting alongside the consolidated portfolio view.
//
// The composite (id, userId) key is enforced at the DB level to prevent one user
// from referencing another user's account via a guessed UUID — all queries must
// scope by both fields.
//
// Balance is the current cash balance maintained by the user (not computed from
// transactions). IsExcluded removes the account from all portfolio aggregations,
// useful for accounts that are tracked for record-keeping only (e.g. a pension
// fund whose performance cannot be influenced by the user).
//
// PlatformId is a nullable FK to a Platform row (broker metadata), omitted here
// because the Platform model is not yet ported to the Go server.
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

// AccountBalance records a point-in-time cash balance snapshot for an account,
// allowing the UI to render a balance history chart independent of transaction
// history. The (accountId, date) unique constraint means only one balance entry
// per calendar day is stored; an upsert on that pair is the expected write pattern.
//
// WHY a separate table: balance history has unbounded growth and different
// retention requirements from the Account row itself, and it is queried by date
// range rather than by account ID alone.
type AccountBalance struct {
	ID        string    `db:"id" json:"id"`
	AccountId string    `db:"accountId" json:"accountId"`
	CreatedAt time.Time `db:"createdAt" json:"createdAt"`
	Date      time.Time `db:"date" json:"date"`
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
	UserId    string    `db:"userId" json:"userId"`
	Value     float64   `db:"value" json:"value"`
}
