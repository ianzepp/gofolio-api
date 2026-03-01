package model

import (
	"encoding/json"
	"time"
)

// Access grants a grantee the ability to view the owner's portfolio, enabling
// the portfolio-sharing feature (e.g. sharing a public link or granting a
// trusted person read-only access).
//
// UserId is the portfolio owner; GranteeUserId is the user being granted access.
// GranteeUserId is nullable to support anonymous (link-based) sharing where the
// viewer does not need a Ghostfolio account.
//
// Permissions is a Postgres enum array (AccessPermission values) stored as []string
// to avoid a custom Postgres scanner. The application layer is responsible for
// interpreting these strings as AccessPermission constants.
//
// Alias lets the owner label a grant (e.g. "shared with my accountant") for
// management purposes; it is not visible to the grantee.
//
// Settings is an opaque JSON object for grant-specific display options (e.g.
// which accounts are visible to the grantee). The schema is defined by the
// NestJS layer and passed through unchanged by the Go server.
//
// TRADE-OFF: storing Permissions as a string slice rather than []AccessPermission
// avoids a custom pgx type but means invalid values are only caught at the
// application layer, not at scan time.
type Access struct {
	ID            string          `db:"id" json:"id"`
	Alias         *string         `db:"alias" json:"alias,omitempty"`
	CreatedAt     time.Time       `db:"createdAt" json:"createdAt"`
	GranteeUserId *string         `db:"granteeUserId" json:"granteeUserId,omitempty"`
	Permissions   []string        `db:"permissions" json:"permissions"` // Postgres enum array
	Settings      json.RawMessage `db:"settings" json:"settings"`
	UpdatedAt     time.Time       `db:"updatedAt" json:"updatedAt"`
	UserId        string          `db:"userId" json:"userId"`
}
