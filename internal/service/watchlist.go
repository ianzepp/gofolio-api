package service

import (
	"context"
	"errors"

	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

var (
	ErrWatchlistItemNotFound = errors.New("watchlist item not found")
	ErrSymbolProfileNotFound = errors.New("symbol profile not found")
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
	err := repo.Add(ctx, userID, dataSource, symbol)
	if errors.Is(err, repository.ErrSymbolProfileNotFound) {
		return ErrSymbolProfileNotFound
	}
	return err
}

// RemoveWatchlistItem removes a symbol from the user's watchlist.
func (s *WatchlistService) RemoveWatchlistItem(ctx context.Context, userID string, dataSource string, symbol string) error {
	repo := repository.NewWatchlistRepository(s.db)
	err := repo.Remove(ctx, userID, dataSource, symbol)
	if errors.Is(err, repository.ErrWatchlistItemNotFound) {
		return ErrWatchlistItemNotFound
	}
	return err
}
