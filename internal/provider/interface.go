// Package provider defines the DataProvider interface and the shared response
// types used by all market-data integrations in the Ghostfolio Go API.
//
// Architecture overview:
//
//	DataProvider (interface)
//	  ├── YahooFinanceProvider   — real-time and historical data via Yahoo Finance APIs
//	  └── ManualProvider         — DB-backed data for user-managed symbols
//
// The DataProviderService (in the service package) holds a registry of
// DataProvider implementations and dispatches quote/historical/search requests
// to the appropriate provider based on the DataSource enum. Providers are
// stateless with respect to business logic — they only know how to talk to their
// upstream data source.
//
// TRADE-OFF: Providers return value types (maps, structs) rather than streaming
// them. For the current scale (≤ 50 symbols per batch) this is fine; a streaming
// approach would be needed for bulk portfolio refreshes.
package provider

import (
	"context"
	"time"

	"github.com/ianzepp/gofolio-api/internal/model"
)

// DataProviderResponse holds a single live-quote result from a data provider.
// MarketState indicates trading session status: "open", "closed", or "delayed"
// (the last is used by manual/DB-backed providers that have no real-time feed).
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

// HistoricalResponse holds a single historical price data point. The map key
// used by callers is a YYYY-MM-DD date string — this type carries only the
// value so the key can be determined by the provider's date iteration logic.
type HistoricalResponse struct {
	MarketPrice float64 `json:"marketPrice"`
}

// DataProviderInfo describes a provider's identity and capability flags.
// IsPremium signals that the provider requires a paid API key or subscription;
// this is surfaced in the admin status dashboard.
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

// AssetProfileIdentifier is the composite key used when requesting profiles or
// historical data for a batch of symbols that may span multiple data sources.
// The DataProviderService uses it to group symbols by provider before dispatch.
type AssetProfileIdentifier struct {
	DataSource model.DataSource
	Symbol     string
}

// DataProvider is the interface every market-data integration must implement.
// Implementations must be safe for concurrent use — the DataProviderService
// may call methods from multiple goroutines when handling batched requests.
//
// GetMaxSymbolsPerRequest controls batching: if non-zero the service will split
// symbol lists into chunks of at most that size before calling GetQuotes.
// Return 0 to indicate no batch-size limit.
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
