package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/provider"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	Date        string  `json:"date"`
	MarketPrice float64 `json:"marketPrice"`
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
func (s *SymbolService) Get(ctx context.Context, dataSource model.DataSource, symbol string, includeHistoricalData bool) (*SymbolItem, error) {
	items := []provider.AssetProfileIdentifier{
		{DataSource: dataSource, Symbol: symbol},
	}

	quotes, err := s.dataProviderService.GetQuotes(ctx, items)
	if err != nil {
		return nil, fmt.Errorf("symbol get quotes: %w", err)
	}

	item := &SymbolItem{
		DataSource: dataSource,
		Symbol:     symbol,
	}

	if q, ok := quotes[symbol]; ok {
		item.Currency = q.Currency
		item.MarketPrice = q.MarketPrice
		item.MarketState = q.MarketState
		item.Name = q.Name
		if q.AssetClass != "" {
			ac := model.AssetClass(q.AssetClass)
			item.AssetClass = &ac
		}
		if q.AssetSubClass != "" {
			asc := model.AssetSubClass(q.AssetSubClass)
			item.AssetSubClass = &asc
		}
	}

	if includeHistoricalData && s.pool != nil {
		repo := repository.NewMarketDataRepository(s.pool)
		data, err := repo.FindBySymbol(ctx, string(dataSource), symbol)
		if err == nil {
			points := make([]MarketDataPoint, 0, len(data))
			for _, md := range data {
				points = append(points, MarketDataPoint{
					Date:        md.Date.Format("2006-01-02"),
					MarketPrice: md.MarketPrice,
				})
			}
			item.HistoricalData = points
		}
	}

	return item, nil
}

// GetForDate returns the market price for a symbol on a specific date.
func (s *SymbolService) GetForDate(ctx context.Context, dataSource model.DataSource, symbol string, date time.Time) (map[string]any, error) {
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)

	historical, err := s.dataProviderService.GetHistorical(ctx, dataSource, symbol, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("symbol get for date: %w", err)
	}

	dateStr := startOfDay.Format("2006-01-02")
	price := 0.0
	if h, ok := historical[dateStr]; ok {
		price = h.MarketPrice
	}

	return map[string]any{
		"marketPrice": price,
	}, nil
}

// Lookup delegates to the data provider search.
func (s *SymbolService) Lookup(ctx context.Context, query string, includeIndices bool) (provider.LookupResponse, error) {
	return s.dataProviderService.Search(ctx, query, includeIndices)
}
