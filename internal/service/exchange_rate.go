package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// ExchangeRateService looks up historical exchange rates from the MarketData
// table. Currency pairs are stored as symbols like "USDEUR" with a known data
// source (typically YAHOO).
type ExchangeRateService struct {
	db *sqlx.DB
}

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
		return nil, fmt.Errorf("invalid date: %w", err)
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

// findRate queries MarketData for the most recent price on or before the given
// date for a specific data source and symbol.
func (s *ExchangeRateService) findRate(ctx context.Context, repo *repository.MarketDataRepository, dataSource string, symbol string, date time.Time) (float64, bool, error) {
	md, err := repo.FindLatestOnOrBeforeDate(ctx, dataSource, symbol, date)
	if err != nil {
		return 0, false, nil // not found is not an error
	}
	return md.MarketPrice, true, nil
}
