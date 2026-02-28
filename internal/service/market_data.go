package service

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketDataService struct {
	pool *pgxpool.Pool
}

func NewMarketDataService(pool *pgxpool.Pool) *MarketDataService {
	return &MarketDataService{pool: pool}
}

type MarketDataDetailsResponse struct {
	AssetProfile model.SymbolProfile `json:"assetProfile"`
	MarketData   []model.MarketData  `json:"marketData"`
}

func (s *MarketDataService) GetBySymbol(ctx context.Context, dataSource string, symbol string) (MarketDataDetailsResponse, error) {
	spRepo := repository.NewSymbolProfileRepository(s.pool)
	mdRepo := repository.NewMarketDataRepository(s.pool)

	profile, err := spRepo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		return MarketDataDetailsResponse{}, fmt.Errorf("find symbol profile: %w", err)
	}

	marketData, err := mdRepo.FindBySymbol(ctx, dataSource, symbol)
	if err != nil {
		return MarketDataDetailsResponse{}, fmt.Errorf("find market data: %w", err)
	}

	if marketData == nil {
		marketData = []model.MarketData{}
	}

	return MarketDataDetailsResponse{
		AssetProfile: profile,
		MarketData:   marketData,
	}, nil
}
