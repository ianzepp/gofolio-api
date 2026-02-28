package model

import (
	"encoding/json"
	"time"
)

// SymbolProfile stores metadata about a financial instrument as sourced from an
// external data provider. It is the central reference record that Orders and
// MarketData rows link to via symbolProfileId.
//
// The (dataSource, symbol) pair is the natural key — the same ticker can exist
// under multiple data sources (e.g. "AAPL" on both YAHOO and GHOSTFOLIO), each
// with its own profile record. The UUID ID is used as the FK to avoid tight
// coupling to a specific provider's symbol scheme.
//
// JSON columns (Countries, Holdings, Sectors, SymbolMapping, ScraperConfiguration)
// are stored as json.RawMessage so that the Go layer can pass them through to
// clients without unmarshalling, preserving provider-specific structure that the
// Go server does not interpret directly.
//
// UserId is non-nil only for user-created custom symbols (DataSourceManual),
// indicating that the profile is private to that user.
type SymbolProfile struct {
	ID                   string           `db:"id" json:"id"`
	AssetClass           *AssetClass      `db:"assetClass" json:"assetClass,omitempty"`
	AssetSubClass        *AssetSubClass   `db:"assetSubClass" json:"assetSubClass,omitempty"`
	Comment              *string          `db:"comment" json:"comment,omitempty"`
	Countries            json.RawMessage  `db:"countries" json:"countries,omitempty"`
	CreatedAt            time.Time        `db:"createdAt" json:"createdAt"`
	Currency             string           `db:"currency" json:"currency"`
	Cusip                *string          `db:"cusip" json:"cusip,omitempty"`
	DataSource           DataSource       `db:"dataSource" json:"dataSource"`
	Figi                 *string          `db:"figi" json:"figi,omitempty"`
	FigiComposite        *string          `db:"figiComposite" json:"figiComposite,omitempty"`
	FigiShareClass       *string          `db:"figiShareClass" json:"figiShareClass,omitempty"`
	Holdings             json.RawMessage  `db:"holdings" json:"holdings,omitempty"`
	IsActive             bool             `db:"isActive" json:"isActive"`
	Isin                 *string          `db:"isin" json:"isin,omitempty"`
	Name                 *string          `db:"name" json:"name,omitempty"`
	ScraperConfiguration json.RawMessage  `db:"scraperConfiguration" json:"scraperConfiguration,omitempty"`
	Sectors              json.RawMessage  `db:"sectors" json:"sectors,omitempty"`
	Symbol               string           `db:"symbol" json:"symbol"`
	SymbolMapping        json.RawMessage  `db:"symbolMapping" json:"symbolMapping,omitempty"`
	UpdatedAt            time.Time        `db:"updatedAt" json:"updatedAt"`
	Url                  *string          `db:"url" json:"url,omitempty"`
	UserId               *string          `db:"userId" json:"userId,omitempty"`
}

// SymbolProfileOverrides stores user-supplied corrections to a SymbolProfile.
// Rather than mutating the provider-sourced SymbolProfile directly (which would
// be overwritten on the next data sync), overrides are stored in a sibling row
// keyed by symbolProfileId and merged at query time.
//
// WHY: this separation lets the system re-fetch authoritative data from providers
// without losing user customisations, and makes it easy to diff what the user
// changed vs. what the provider says.
//
// TRADE-OFF: callers must always merge SymbolProfile + SymbolProfileOverrides
// before returning data to clients; forgetting to merge is a correctness bug.
type SymbolProfileOverrides struct {
	SymbolProfileId string          `db:"symbolProfileId" json:"symbolProfileId"`
	AssetClass      *AssetClass     `db:"assetClass" json:"assetClass,omitempty"`
	AssetSubClass   *AssetSubClass  `db:"assetSubClass" json:"assetSubClass,omitempty"`
	Countries       json.RawMessage `db:"countries" json:"countries,omitempty"`
	Holdings        json.RawMessage `db:"holdings" json:"holdings,omitempty"`
	Name            *string         `db:"name" json:"name,omitempty"`
	Sectors         json.RawMessage `db:"sectors" json:"sectors,omitempty"`
	UpdatedAt       time.Time       `db:"updatedAt" json:"updatedAt"`
	Url             *string         `db:"url" json:"url,omitempty"`
}
