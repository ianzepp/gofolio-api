package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/provider"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSymbolNotFound = errors.New("symbol not found")

// SymbolItem represents the response for a single symbol lookup.
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

// MarketDataPoint is a single date+price for historical data.
type MarketDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// SymbolService provides symbol-level operations.
type SymbolService struct {
	pool                *pgxpool.Pool
	dataProviderService *DataProviderService
}

// NewSymbolService creates a new SymbolService.
func NewSymbolService(pool *pgxpool.Pool, dps *DataProviderService) *SymbolService {
	return &SymbolService{pool: pool, dataProviderService: dps}
}

// Get returns a quote (and optionally historical data) for a symbol.
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
	if includeHistoricalDataDays > 0 && s.pool != nil {
		repo := repository.NewMarketDataRepository(s.pool)
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

// GetForDate returns the market price for a symbol on a specific date.
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

// Lookup delegates to the data provider search.
func (s *SymbolService) Lookup(ctx context.Context, query string, includeIndices bool) (provider.LookupResponse, error) {
	return s.dataProviderService.Search(ctx, query, includeIndices)
}
