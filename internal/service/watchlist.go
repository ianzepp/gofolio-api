package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// WatchlistService manages a user's symbol watchlist.
type WatchlistService struct {
	db *sqlx.DB
}

func NewWatchlistService(db *sqlx.DB) *WatchlistService {
	return &WatchlistService{db: db}
}

// GetWatchlist returns the user's watched symbols.
func (s *WatchlistService) GetWatchlist(ctx context.Context, userID string) ([]repository.WatchlistItem, error) {
	repo := repository.NewWatchlistRepository(s.db)
	items, err := repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []repository.WatchlistItem{}
	}
	return items, nil
}

// AddWatchlistItem adds a symbol to the user's watchlist.
func (s *WatchlistService) AddWatchlistItem(ctx context.Context, userID string, dataSource string, symbol string) error {
	repo := repository.NewWatchlistRepository(s.db)
	return repo.Add(ctx, userID, dataSource, symbol)
}

// RemoveWatchlistItem removes a symbol from the user's watchlist.
func (s *WatchlistService) RemoveWatchlistItem(ctx context.Context, userID string, dataSource string, symbol string) error {
	repo := repository.NewWatchlistRepository(s.db)
	return repo.Remove(ctx, userID, dataSource, symbol)
}
