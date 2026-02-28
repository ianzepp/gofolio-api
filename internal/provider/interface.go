package provider

import (
	"context"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
)

// DataProviderResponse holds a single quote result from a data provider.
type DataProviderResponse struct {
	Currency       string  `json:"currency"`
	DataSource     string  `json:"dataSource"`
	MarketPrice    float64 `json:"marketPrice"`
	MarketState    string  `json:"marketState"` // "open", "closed", "delayed"
	Name           string  `json:"name,omitempty"`
	Symbol         string  `json:"symbol,omitempty"`
	AssetClass     string  `json:"assetClass,omitempty"`
	AssetSubClass  string  `json:"assetSubClass,omitempty"`
}

// HistoricalResponse holds a single historical data point.
type HistoricalResponse struct {
	MarketPrice float64 `json:"marketPrice"`
}

// DataProviderInfo describes a provider's capabilities.
type DataProviderInfo struct {
	IsPremium        bool   `json:"isPremium"`
	Name             string `json:"name"`
	URL              string `json:"url"`
	AssetProfileCount int   `json:"assetProfileCount,omitempty"`
}

// LookupItem is a single search result entry.
type LookupItem struct {
	AssetClass    string           `json:"assetClass,omitempty"`
	AssetSubClass string           `json:"assetSubClass,omitempty"`
	Currency      string           `json:"currency"`
	DataSource    model.DataSource `json:"dataSource"`
	Name          string           `json:"name"`
	Symbol        string           `json:"symbol"`
}

// LookupResponse wraps search results.
type LookupResponse struct {
	Items []LookupItem `json:"items"`
}

// AssetProfileIdentifier identifies a symbol within a specific data source.
type AssetProfileIdentifier struct {
	DataSource model.DataSource
	Symbol     string
}

// DataProvider is the interface every market data provider must implement.
type DataProvider interface {
	// CanHandle returns true if the provider can handle the given symbol.
	CanHandle(symbol string) bool

	// GetName returns the DataSource enum for this provider.
	GetName() model.DataSource

	// GetTestSymbol returns a symbol that can be used to test this provider.
	GetTestSymbol() string

	// GetMaxSymbolsPerRequest returns the max batch size. 0 means unlimited.
	GetMaxSymbolsPerRequest() int

	// GetDataProviderInfo returns metadata about this provider.
	GetDataProviderInfo() DataProviderInfo

	// GetQuotes fetches live quotes for the given symbols.
	GetQuotes(ctx context.Context, symbols []string) (map[string]DataProviderResponse, error)

	// GetHistorical fetches historical daily prices.
	GetHistorical(ctx context.Context, symbol string, from, to time.Time) (map[string]HistoricalResponse, error)

	// GetDividends fetches dividend history.
	GetDividends(ctx context.Context, symbol string, from, to time.Time) (map[string]HistoricalResponse, error)

	// GetAssetProfile fetches asset profile information.
	GetAssetProfile(ctx context.Context, symbol string) (*model.SymbolProfile, error)

	// Search performs a symbol search.
	Search(ctx context.Context, query string, includeIndices bool) (LookupResponse, error)
}
