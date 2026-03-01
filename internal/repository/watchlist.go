// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ianzepp/gofolio-api/internal/db"
	"github.com/jmoiron/sqlx"
)

var (
	ErrWatchlistItemNotFound = errors.New("watchlist item not found")
	ErrSymbolProfileNotFound = errors.New("symbol profile not found")
)

// WatchlistItem is a lightweight projection of a SymbolProfile used for
// watchlist display. It avoids loading the full SymbolProfile columns.
type WatchlistItem struct {
	DataSource string  `db:"dataSource" json:"dataSource"`
	Symbol     string  `db:"symbol" json:"symbol"`
	Name       *string `db:"name" json:"name"`
}

// WatchlistRepository queries the _UserWatchlist M:M join table between User
// and SymbolProfile.
type WatchlistRepository struct {
	db db.DBTX
}

func NewWatchlistRepository(db db.DBTX) *WatchlistRepository {
	return &WatchlistRepository{db: db}
}

// FindByUserID returns all watched symbols for a user, ordered by name.
func (r *WatchlistRepository) FindByUserID(ctx context.Context, userID string) ([]WatchlistItem, error) {
	var items []WatchlistItem
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT sp."dataSource", sp.symbol, sp.name
		 FROM "SymbolProfile" sp
		 JOIN "_UserWatchlist" uw ON uw."A" = sp.id
		 WHERE uw."B" = $1
		 ORDER BY sp.name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query watchlist: %w", err)
	}
	return items, nil
}

// Add connects a SymbolProfile to a user's watchlist. Uses ON CONFLICT DO
// NOTHING so adding the same symbol twice is a no-op.
func (r *WatchlistRepository) Add(ctx context.Context, userID string, dataSource string, symbol string) error {
	var symbolProfileExists bool
	if err := sqlx.GetContext(ctx, r.db, &symbolProfileExists,
		`SELECT EXISTS (
			SELECT 1 FROM "SymbolProfile"
			WHERE "dataSource" = $1 AND symbol = $2
		)`, dataSource, symbol); err != nil {
		return fmt.Errorf("check symbol profile for watchlist: %w", err)
	}
	if !symbolProfileExists {
		return ErrSymbolProfileNotFound
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO "_UserWatchlist" ("A", "B")
		 SELECT sp.id, $1
		 FROM "SymbolProfile" sp
		 WHERE sp."dataSource" = $2 AND sp.symbol = $3
		 ON CONFLICT DO NOTHING`, userID, dataSource, symbol)
	if err != nil {
		return fmt.Errorf("add watchlist item: %w", err)
	}
	return nil
}

// Remove disconnects a SymbolProfile from a user's watchlist.
func (r *WatchlistRepository) Remove(ctx context.Context, userID string, dataSource string, symbol string) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM "_UserWatchlist"
		 WHERE "A" = (
		   SELECT id FROM "SymbolProfile"
		   WHERE "dataSource" = $1 AND symbol = $2
		 ) AND "B" = $3`, dataSource, symbol, userID)
	if err != nil {
		return fmt.Errorf("remove watchlist item: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("remove watchlist rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrWatchlistItemNotFound
	}
	return nil
}
