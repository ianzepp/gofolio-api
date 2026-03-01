package model

import "time"

// User represents a Ghostfolio user account. It is the root ownership entity:
// Accounts, Orders, Access grants, and Settings all hang off a User record.
//
// AccessToken holds the short-lived bearer token issued after login; it is
// stored in the DB so that logout can be enforced server-side by clearing it.
// AuthChallenge holds a one-time nonce used during WebAuthn or anonymous-login
// challenge/response flows; it is cleared once authentication succeeds.
//
// SAFETY: AccessToken and AuthChallenge must never be included in API responses
// returned to other users. Callers should project only the fields they need.
type User struct {
	ID            string    `db:"id" json:"id"`
	AccessToken   *string   `db:"accessToken" json:"-"`
	AuthChallenge *string   `db:"authChallenge" json:"-"`
	CreatedAt     time.Time `db:"createdAt" json:"createdAt"`
	Provider      Provider  `db:"provider" json:"provider"`
	Role          Role      `db:"role" json:"role"`
	ThirdPartyId  *string   `db:"thirdPartyId" json:"thirdPartyId,omitempty"`
	UpdatedAt     time.Time `db:"updatedAt" json:"updatedAt"`
}

// Settings stores the user's application preferences as an opaque JSON blob.
// Storing preferences as JSON rather than columns avoids schema migrations
// for every new UI setting, at the cost of losing type safety at the DB layer.
// The Settings field is a *string (nullable) rather than json.RawMessage so
// that a missing row and an explicit null are both represented as nil.
//
// WHY a separate table: Settings has a different write pattern (updated on every
// preference change) than User (updated rarely), reducing row churn on the User
// table and simplifying cache invalidation.
type Settings struct {
	UserId    string    `db:"userId" json:"userId"`
	Settings  *string   `db:"settings" json:"settings,omitempty"` // JSON string
	UpdatedAt time.Time `db:"updatedAt" json:"updatedAt"`
}

// Analytics tracks aggregate usage metrics for a single user. This is used by
// the Ghostfolio admin dashboard to understand activity levels and to enforce
// rate limits on premium data provider requests (DataProviderGhostfolioDailyRequests).
//
// NOTE: ActivityCount is incremented on Order mutations, not on read operations,
// so it reflects portfolio activity rather than login frequency.
// LastRequestAt records the last time the user triggered a data provider fetch,
// enabling daily quota reset logic.
type Analytics struct {
	UserId                              string    `db:"userId" json:"userId"`
	ActivityCount                       int       `db:"activityCount" json:"activityCount"`
	Country                             *string   `db:"country" json:"country,omitempty"`
	DataProviderGhostfolioDailyRequests int       `db:"dataProviderGhostfolioDailyRequests" json:"dataProviderGhostfolioDailyRequests"`
	LastRequestAt                       time.Time `db:"lastRequestAt" json:"lastRequestAt"`
	UpdatedAt                           time.Time `db:"updatedAt" json:"updatedAt"`
}
