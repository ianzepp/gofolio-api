package service

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// MarketDataService retrieves historical market data paired with the asset
// profile for a given symbol. It is used by the admin market data detail
// endpoint where a full SymbolProfile (not just a name) is required, unlike
// AssetService which degrades to a synthetic profile for unknown symbols.
type MarketDataService struct {
	db *sqlx.DB
}

// NewMarketDataService constructs a MarketDataService backed by the given pool.
func NewMarketDataService(db *sqlx.DB) *MarketDataService {
	return &MarketDataService{db: db}
}

// MarketDataDetailsResponse bundles the full SymbolProfile with all stored
// MarketData rows for a symbol, giving the admin detail page everything it
// needs to render metadata and a price history chart.
type MarketDataDetailsResponse struct {
	AssetProfile model.SymbolProfile `json:"assetProfile"`
	MarketData   []model.MarketData  `json:"marketData"`
}

// GetBySymbol returns the asset profile and all market data for a symbol.
// Unlike AssetService.GetAsset, this method requires a SymbolProfile to exist
// and returns an error if none is found, since it is only called from admin
// flows where the symbol is known to be profiled.
func (s *MarketDataService) GetBySymbol(ctx context.Context, dataSource string, symbol string) (MarketDataDetailsResponse, error) {
	spRepo := repository.NewSymbolProfileRepository(s.db)
	mdRepo := repository.NewMarketDataRepository(s.db)

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
