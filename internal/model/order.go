package model

import "time"

// Order represents any portfolio activity — the name "Order" is inherited from
// the original Prisma schema but the type covers the full range of OrderType
// values (BUY, SELL, DIVIDEND, FEE, INTEREST, ITEM, LIABILITY, LIQUIDITY).
//
// SymbolProfileId links every order to a symbol, even for non-traded types such
// as LIQUIDITY, which reference a currency pseudo-symbol. This keeps the
// portfolio calculation engine uniform: every activity has an associated price
// series regardless of type.
//
// AccountId and AccountUserId are nullable because orders can exist without an
// account (e.g. imported activities from a broker export before the user has
// configured accounts). AccountUserId mirrors the account's userId to allow an
// efficient composite-FK check without a join.
//
// IsDraft marks orders that have been imported but not confirmed by the user;
// draft orders are excluded from portfolio performance calculations.
//
// EDGE: Quantity and UnitPrice are float64. For very high-precision instruments
// (e.g. fractional crypto) this can introduce rounding drift over many rows.
// The original NestJS app uses Decimal.js on the client side to compensate.
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
