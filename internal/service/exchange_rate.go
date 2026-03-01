package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// ExchangeRateService looks up historical exchange rates from the MarketData
// table. Currency pairs are stored as symbols like "USDEUR" with a known data
// source (typically YAHOO).
type ExchangeRateService struct {
	db *sqlx.DB
}

var ErrInvalidExchangeRateDate = errors.New("invalid exchange-rate date")

func NewExchangeRateService(db *sqlx.DB) *ExchangeRateService {
	return &ExchangeRateService{db: db}
}

// ExchangeRateResponse is the JSON shape returned by GET /exchange-rate.
type ExchangeRateResponse struct {
	MarketPrice float64 `json:"marketPrice"`
}

// GetExchangeRate returns the exchange rate for a currency-pair symbol on or
// before the given date. The symbol format matches the Ghostfolio convention
// (e.g. "USDEUR"). Returns the response and a bool indicating whether a rate
// was found.
func (s *ExchangeRateService) GetExchangeRate(ctx context.Context, symbol string, dateString string) (*ExchangeRateResponse, error) {
	date, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidExchangeRateDate, err)
	}

	repo := repository.NewMarketDataRepository(s.db)

	// Try YAHOO first — this is the primary data source for forex rates in
	// the legacy NestJS codebase.
	rate, found, err := s.findRate(ctx, repo, "YAHOO", symbol, date)
	if err != nil {
		return nil, err
	}
	if found {
		return &ExchangeRateResponse{MarketPrice: rate}, nil
	}

	// Try MANUAL as fallback.
	rate, found, err = s.findRate(ctx, repo, "MANUAL", symbol, date)
	if err != nil {
		return nil, err
	}
	if found {
		return &ExchangeRateResponse{MarketPrice: rate}, nil
	}

	return nil, nil // not found
}

// GetExchangeRatesByCurrency returns historical exchange rates for multiple
// source currencies against a target currency over a date range. The result is
// keyed as map[pair][date] = rate, where pair is e.g. "USDCHF".
// Rates are forward-filled: if no market data exists for a given date, the
// most recent prior rate is carried forward.
func (s *ExchangeRateService) GetExchangeRatesByCurrency(
	ctx context.Context,
	currencies []string,
	targetCurrency string,
	startDate, endDate time.Time,
) (map[string]map[string]float64, error) {
	result := make(map[string]map[string]float64)

	repo := repository.NewMarketDataRepository(s.db)

	for _, sourceCurrency := range currencies {
		pair := sourceCurrency + targetCurrency

		if _, exists := result[pair]; exists {
			continue // already computed
		}

		rates := make(map[string]float64)

		if sourceCurrency == targetCurrency {
			// Same currency: all rates are 1.0
			for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
				dateStr := d.Format("2006-01-02")
				rates[dateStr] = 1.0
			}
			result[pair] = rates
			continue
		}

		// Query market data for this FX pair
		symbol := sourceCurrency + targetCurrency

		// Try YAHOO first, then MANUAL
		var marketData []model.MarketData
		var err error

		items := []struct{ DataSource, Symbol string }{
			{DataSource: "YAHOO", Symbol: symbol},
		}
		marketData, err = repo.FindBySymbolsAndDateRange(ctx, items, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("query exchange rates for %s: %w", symbol, err)
		}

		if len(marketData) == 0 {
			items[0].DataSource = "MANUAL"
			marketData, err = repo.FindBySymbolsAndDateRange(ctx, items, startDate, endDate)
			if err != nil {
				return nil, fmt.Errorf("query exchange rates for %s (MANUAL): %w", symbol, err)
			}
		}

		// Build date->rate map from market data
		mdByDate := make(map[string]float64)
		for _, md := range marketData {
			dateStr := md.Date.Format("2006-01-02")
			mdByDate[dateStr] = md.MarketPrice
		}

		// Backward-fill rates across the date range (matching legacy behavior).
		// Legacy seeds previousExchangeRate from the last date, defaulting to 1.
		lastDateStr := endDate.Format("2006-01-02")
		lastRate := 1.0
		if r, ok := mdByDate[lastDateStr]; ok {
			lastRate = r
		}
		for d := endDate; !d.Before(startDate); d = d.AddDate(0, 0, -1) {
			dateStr := d.Format("2006-01-02")
			if r, ok := mdByDate[dateStr]; ok {
				lastRate = r
				rates[dateStr] = r
			} else {
				rates[dateStr] = lastRate
			}
		}

		result[pair] = rates
	}

	return result, nil
}

// findRate queries MarketData for the most recent price on or before the given
// date for a specific data source and symbol.
func (s *ExchangeRateService) findRate(ctx context.Context, repo *repository.MarketDataRepository, dataSource string, symbol string, date time.Time) (float64, bool, error) {
	md, err := repo.FindLatestOnOrBeforeDate(ctx, dataSource, symbol, date)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil // not found is not an error
		}
		return 0, false, err
	}
	return md.MarketPrice, true, nil
}
