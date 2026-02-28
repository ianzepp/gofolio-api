package model

import "time"

// AssetProfileResolution maps symbols between data sources.
// Unique: (dataSourceOrigin, symbolOrigin)
type AssetProfileResolution struct {
	ID               string     `db:"id" json:"id"`
	CreatedAt        time.Time  `db:"createdAt" json:"createdAt"`
	Currency         string     `db:"currency" json:"currency"`
	DataSourceOrigin DataSource `db:"dataSourceOrigin" json:"dataSourceOrigin"`
	DataSourceTarget DataSource `db:"dataSourceTarget" json:"dataSourceTarget"`
	RequestCount     int        `db:"requestCount" json:"requestCount"`
	SymbolOrigin     string     `db:"symbolOrigin" json:"symbolOrigin"`
	SymbolTarget     string     `db:"symbolTarget" json:"symbolTarget"`
	UpdatedAt        time.Time  `db:"updatedAt" json:"updatedAt"`
}
