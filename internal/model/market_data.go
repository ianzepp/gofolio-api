package model

import "time"

// MarketData stores historical market prices.
// Unique: (dataSource, date, symbol)
type MarketData struct {
	ID          string          `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"createdAt" json:"createdAt"`
	DataSource  DataSource      `db:"dataSource" json:"dataSource"`
	Date        time.Time       `db:"date" json:"date"`
	MarketPrice float64         `db:"marketPrice" json:"marketPrice"`
	State       MarketDataState `db:"state" json:"state"`
	Symbol      string          `db:"symbol" json:"symbol"`
}
