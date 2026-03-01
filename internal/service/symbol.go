package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/provider"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// ErrSymbolNotFound is returned by Get when the provider returns no quote for
// the requested symbol. Callers should translate this to a 404 HTTP response.
var ErrSymbolNotFound = errors.New("symbol not found")

// SymbolItem is the response shape for a single symbol lookup. It combines live
// quote data from the provider with optional historical prices from the DB,
// giving clients everything they need to render a symbol detail page.
type SymbolItem struct {
	AssetClass     *model.AssetClass    `json:"assetClass,omitempty"`
	AssetSubClass  *model.AssetSubClass `json:"assetSubClass,omitempty"`
	Currency       string               `json:"currency"`
	DataSource     model.DataSource     `json:"dataSource"`
	MarketPrice    float64              `json:"marketPrice"`
	MarketState    string               `json:"marketState,omitempty"`
	Name           string               `json:"name,omitempty"`
	Symbol         string               `json:"symbol"`
	HistoricalData []MarketDataPoint    `json:"historicalData,omitempty"`
}

// MarketDataPoint is a single date+price pair in the historical series returned
// inside a SymbolItem. The Date field uses ISO 8601 with milliseconds to match
// the NestJS serialization format expected by the frontend charting library.
type MarketDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// SymbolService exposes symbol-level read operations: live quote lookup,
// historical price retrieval, and symbol search. It is a thin coordinator
// between the HTTP handler and DataProviderService, adding only the
// historical-data DB enrichment step that providers cannot supply alone.
type SymbolService struct {
	db                  *sqlx.DB
	dataProviderService *DataProviderService
}

// NewSymbolService creates a new SymbolService.
func NewSymbolService(db *sqlx.DB, dps *DataProviderService) *SymbolService {
	return &SymbolService{db: db, dataProviderService: dps}
}

// Get returns a live quote for a symbol, optionally enriched with historical
// price data from the local DB going back includeHistoricalDataDays days.
// Returns ErrSymbolNotFound if the provider returns no quote or a negative price.
//
// EDGE: A negative MarketPrice is treated as "not found" rather than a valid
// quote, because some providers return -1 as a sentinel for unavailable data.
func (s *SymbolService) Get(ctx context.Context, dataSource model.DataSource, symbol string, includeHistoricalDataDays int) (*SymbolItem, error) {
	items := []provider.AssetProfileIdentifier{
		{DataSource: dataSource, Symbol: symbol},
	}

	quotes, err := s.dataProviderService.GetQuotes(ctx, items)
	if err != nil {
		return nil, fmt.Errorf("symbol get quotes: %w", err)
	}

	q, ok := quotes[symbol]
	if !ok || q.MarketPrice < 0 {
		return nil, ErrSymbolNotFound
	}

	item := &SymbolItem{
		DataSource:  dataSource,
		Symbol:      symbol,
		Currency:    q.Currency,
		MarketPrice: q.MarketPrice,
		MarketState: q.MarketState,
		Name:        q.Name,
	}

	if q.AssetClass != "" {
		ac := model.AssetClass(q.AssetClass)
		item.AssetClass = &ac
	}
	if q.AssetSubClass != "" {
		asc := model.AssetSubClass(q.AssetSubClass)
		item.AssetSubClass = &asc
	}
	if includeHistoricalDataDays > 0 && s.db != nil {
		repo := repository.NewMarketDataRepository(s.db)
		from := time.Now().UTC().AddDate(0, 0, -includeHistoricalDataDays)
		to := time.Now().UTC()
		data, err := repo.FindBySymbolAndDateRange(ctx, string(dataSource), symbol, from, to)
		if err == nil {
			points := make([]MarketDataPoint, 0, len(data))
			for _, md := range data {
				points = append(points, MarketDataPoint{
					Date:  md.Date.UTC().Format("2006-01-02T15:04:05.000Z"),
					Value: md.MarketPrice,
				})
			}
			item.HistoricalData = points
		}
	}

	return item, nil
}

// GetForDate returns the market price for a symbol on a specific calendar date
// by fetching directly from the provider (not the DB cache). If no price is
// available for the given date, an empty object is returned to match the legacy
// NestJS parity contract — callers should treat an absent "marketPrice" key as
// a not-found condition rather than expecting an error.
func (s *SymbolService) GetForDate(ctx context.Context, dataSource model.DataSource, symbol string, date time.Time) (map[string]any, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	items := []provider.AssetProfileIdentifier{
		{DataSource: dataSource, Symbol: symbol},
	}
	historical, err := s.dataProviderService.GetHistoricalRaw(ctx, items, startOfDay, startOfDay)
	if err != nil {
		return nil, fmt.Errorf("symbol get for date: %w", err)
	}

	dateStr := startOfDay.Format("2006-01-02")
	if bySymbol, ok := historical[symbol]; ok {
		if h, ok := bySymbol[dateStr]; ok {
			return map[string]any{
				"marketPrice": h.MarketPrice,
			}, nil
		}
	}

	// Legacy parity: undefined marketPrice serializes as an empty object.
	return map[string]any{}, nil
}

// Lookup delegates symbol search to the DataProviderService, which fans out
// the query to all registered providers and merges results.
func (s *SymbolService) Lookup(ctx context.Context, query string, includeIndices bool) (provider.LookupResponse, error) {
	return s.dataProviderService.Search(ctx, query, includeIndices)
}
