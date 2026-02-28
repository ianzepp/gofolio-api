package model

import (
	"encoding/json"
	"time"
)

// SymbolProfile stores metadata about a financial instrument.
// Unique: (dataSource, symbol)
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

// SymbolProfileOverrides stores per-symbol customizations.
// PK: symbolProfileId
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
