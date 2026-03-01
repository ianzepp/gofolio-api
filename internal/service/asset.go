package service

import (
	"context"
	"errors"
	"fmt"

	"database/sql"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// AssetService retrieves the combined asset profile + market data view used by
// the admin market data detail page. It differs from MarketDataService in that
// it degrades gracefully when no SymbolProfile exists, synthesising a minimal
// profile from the symbol string itself (used for currency pairs).
type AssetService struct {
	db *sqlx.DB
}

// NewAssetService constructs an AssetService backed by the given database.
func NewAssetService(db *sqlx.DB) *AssetService {
	return &AssetService{db: db}
}

// AssetResponse bundles a simplified asset profile with all stored market data
// for the symbol. The profile is a plain map rather than a SymbolProfile struct
// so that only the fields relevant to the asset detail page are serialised,
// maintaining parity with the NestJS endpoint shape.
type AssetResponse struct {
	AssetProfile map[string]any     `json:"assetProfile"`
	MarketData   []model.MarketData `json:"marketData"`
}

// GetAsset returns the asset profile and full market data history for a symbol.
// When no SymbolProfile is found (pgx.ErrNoRows), a synthetic profile is built
// from the symbol string. This handles currency pair symbols (e.g. "GBPUSD")
// that exist in MarketData but may not have been explicitly profiled.
//
// EDGE: getCurrencyFromSymbol strips the base currency to extract the quote
// currency code, then isCurrencyCode validates it. An unrecognised suffix
// falls back to "-" so that the asset page still renders rather than erroring.
func (s *AssetService) GetAsset(ctx context.Context, dataSource string, symbol string) (AssetResponse, error) {
	spRepo := repository.NewSymbolProfileRepository(s.db)
	mdRepo := repository.NewMarketDataRepository(s.db)

	profile, err := spRepo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return AssetResponse{}, fmt.Errorf("find symbol profile: %w", err)
		}

		currency := "-"
		if candidate := getCurrencyFromSymbol(symbol); isCurrencyCode(candidate) {
			currency = candidate
		}

		profile = model.SymbolProfile{
			Currency:   currency,
			DataSource: model.DataSource(dataSource),
			IsActive:   true,
			Symbol:     symbol,
		}
	}

	marketData, err := mdRepo.FindBySymbol(ctx, dataSource, symbol)
	if err != nil {
		return AssetResponse{}, fmt.Errorf("find market data: %w", err)
	}

	if marketData == nil {
		marketData = []model.MarketData{}
	}

	// Asset endpoint returns a simplified profile
	assetProfile := map[string]any{
		"dataSource": profile.DataSource,
		"name":       profile.Name,
		"symbol":     profile.Symbol,
	}

	return AssetResponse{
		AssetProfile: assetProfile,
		MarketData:   marketData,
	}, nil
}
