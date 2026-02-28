package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetService struct {
	pool *pgxpool.Pool
}

func NewAssetService(pool *pgxpool.Pool) *AssetService {
	return &AssetService{pool: pool}
}

type AssetResponse struct {
	AssetProfile map[string]any     `json:"assetProfile"`
	MarketData   []model.MarketData `json:"marketData"`
}

func (s *AssetService) GetAsset(ctx context.Context, dataSource string, symbol string) (AssetResponse, error) {
	spRepo := repository.NewSymbolProfileRepository(s.pool)
	mdRepo := repository.NewMarketDataRepository(s.pool)

	profile, err := spRepo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
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
