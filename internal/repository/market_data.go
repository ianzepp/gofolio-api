// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MarketDataRepository handles persistence for daily price points scraped or
// fetched from external data providers. Each row represents one closing price
// for one symbol on one day from one source; the unique constraint on
// (dataSource, date, symbol) is used for idempotent upserts so that the
// data-collection pipeline can re-run without creating duplicates.
type MarketDataRepository struct {
	db db.DBTX
}

// NewMarketDataRepository constructs a MarketDataRepository backed by the
// provided database handle.
func NewMarketDataRepository(db db.DBTX) *MarketDataRepository {
	return &MarketDataRepository{db: db}
}

// marketDataColumns is the canonical SELECT list for MarketData rows.
const marketDataColumns = `id, "createdAt", "dataSource", date,
	"marketPrice", state, symbol`

// FindBySymbol returns the full price history for a symbol from a given data
// source, ordered newest-first. The result set may be large for actively-
// tracked securities — callers that only need the latest price should prefer
// FindLatestByDataSourceAndSymbol.
func (r *MarketDataRepository) FindBySymbol(ctx context.Context, dataSource string, symbol string) ([]model.MarketData, error) {
	var items []model.MarketData
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2
		 ORDER BY date DESC`, dataSource, symbol)
	if err != nil {
		return nil, fmt.Errorf("query market data: %w", err)
	}
	return items, nil
}

// DeleteByDataSourceAndSymbol removes all market data rows for a given symbol.
func (r *MarketDataRepository) DeleteByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM "MarketData" WHERE "dataSource" = $1 AND symbol = $2`, dataSource, symbol)
	if err != nil {
		return fmt.Errorf("delete market data by symbol: %w", err)
	}
	return nil
}

// FindBySymbolAndDateRange returns price points for a symbol within a closed
// date interval [from, to], ordered chronologically ascending. This ordering
// is important for time-series calculations (TWR, MWRR) that must process
// data in forward chronological order.
func (r *MarketDataRepository) FindBySymbolAndDateRange(ctx context.Context, dataSource string, symbol string, from time.Time, to time.Time) ([]model.MarketData, error) {
	var items []model.MarketData
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2
		   AND date >= $3 AND date <= $4
		 ORDER BY date ASC`, dataSource, symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("query market data range: %w", err)
	}
	return items, nil
}

// CountByDataSourceAndSymbol returns the number of stored price points for a
// given symbol and data source. Used to decide whether historical backfill is
// needed before serving portfolio calculations.
func (r *MarketDataRepository) CountByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2`, dataSource, symbol).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count market data: %w", err)
	}
	return count, nil
}

// FindLatestByDataSourceAndSymbol retrieves the most recent price record for
// a single symbol. The LIMIT 1 with ORDER BY date DESC avoids a full scan
// when a suitable index on (dataSource, symbol, date DESC) exists.
func (r *MarketDataRepository) FindLatestByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (model.MarketData, error) {
	var item model.MarketData
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2
		 ORDER BY date DESC
		 LIMIT 1`, dataSource, symbol)
	if err != nil {
		return model.MarketData{}, fmt.Errorf("query latest market data: %w", err)
	}
	return item, nil
}

// FindLatestBySymbols retrieves the latest price record for each symbol in the
// provided list, all from the same data source. Returns an empty slice without
// hitting the database when symbols is empty.
//
// PERF: uses a ROW_NUMBER() window function to select the most recent row per
// symbol, which is portable across database engines.
func (r *MarketDataRepository) FindLatestBySymbols(ctx context.Context, dataSource string, symbols []string) ([]model.MarketData, error) {
	if len(symbols) == 0 {
		// EDGE: early-return avoids sending an empty IN list to the database,
		// which would produce an unhelpful empty result and waste a round-trip.
		return []model.MarketData{}, nil
	}
	query, args, err := sqlx.In(
		`SELECT id, "createdAt", "dataSource", date, "marketPrice", state, symbol
		 FROM (
		   SELECT id, "createdAt", "dataSource", date, "marketPrice", state, symbol,
		          ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY date DESC) AS rn
		   FROM "MarketData"
		   WHERE "dataSource" = ? AND symbol IN (?)
		 ) sub
		 WHERE rn = 1`, dataSource, symbols)
	if err != nil {
		return nil, fmt.Errorf("build latest market data query: %w", err)
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)
	var items []model.MarketData
	err = sqlx.SelectContext(ctx, r.db, &items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query latest market data by symbols: %w", err)
	}
	return items, nil
}

// UpsertMarketPrice inserts a new price point or updates an existing one
// identified by the unique constraint on (dataSource, date, symbol). This is
// the idempotent write path used by the data-provider scraper pipeline — it
// is safe to call repeatedly with the same inputs.
// NOTE: the ID is generated in Go via uuid.New() so the query is portable
// across database engines.
func (r *MarketDataRepository) UpsertMarketPrice(ctx context.Context, dataSource string, symbol string, date time.Time, price float64, state model.MarketDataState) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO "MarketData" (id, "dataSource", date, "marketPrice", state, symbol)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT ("dataSource", date, symbol) DO UPDATE
		   SET "marketPrice" = EXCLUDED."marketPrice",
		       state = EXCLUDED.state`,
		uuid.New().String(), dataSource, date, price, state, symbol)
	if err != nil {
		return fmt.Errorf("upsert market price: %w", err)
	}
	return nil
}

// FindLatestOnOrBeforeDate retrieves the most recent price record for a symbol
// on or before the given date. Used by exchange rate lookups to find the
// closest available rate when an exact date match may not exist.
func (r *MarketDataRepository) FindLatestOnOrBeforeDate(ctx context.Context, dataSource string, symbol string, date time.Time) (model.MarketData, error) {
	var item model.MarketData
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2 AND date <= $3
		 ORDER BY date DESC
		 LIMIT 1`, dataSource, symbol, date)
	if err != nil {
		return model.MarketData{}, fmt.Errorf("query latest market data on or before date: %w", err)
	}
	return item, nil
}

// FindBySymbolsAndDateRange returns price points for multiple symbols within a
// date range, ordered chronologically ascending. Used by the portfolio calculator
// to batch-fetch market data for all holdings at once.
func (r *MarketDataRepository) FindBySymbolsAndDateRange(
	ctx context.Context,
	items []struct{ DataSource, Symbol string },
	from, to time.Time,
) ([]model.MarketData, error) {
	if len(items) == 0 {
		return []model.MarketData{}, nil
	}

	// Build (dataSource, symbol) pairs for the IN clause
	type pair struct {
		DataSource string `db:"ds"`
		Symbol     string `db:"sym"`
	}
	pairs := make([]pair, len(items))
	for i, item := range items {
		pairs[i] = pair{DataSource: item.DataSource, Symbol: item.Symbol}
	}

	// Use OR conditions for each pair since sqlx.In doesn't support tuple IN
	query := `SELECT ` + marketDataColumns + `
		 FROM "MarketData"
		 WHERE date >= $1 AND date <= $2 AND (`
	args := []any{from, to}
	for i, p := range pairs {
		if i > 0 {
			query += " OR "
		}
		query += fmt.Sprintf(`("dataSource" = $%d AND symbol = $%d)`, len(args)+1, len(args)+2)
		args = append(args, p.DataSource, p.Symbol)
	}
	query += `) ORDER BY date ASC`

	var result []model.MarketData
	err := sqlx.SelectContext(ctx, r.db, &result, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query market data batch: %w", err)
	}
	return result, nil
}

// Create inserts a single market data row with a caller-supplied ID and returns
// the full persisted record. Unlike UpsertMarketPrice, this will fail with a
// unique-constraint error if a record for the same (dataSource, date, symbol)
// already exists — use UpsertMarketPrice for idempotent writes.
func (r *MarketDataRepository) Create(ctx context.Context, md model.MarketData) (model.MarketData, error) {
	var item model.MarketData
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "MarketData" (id, "dataSource", date, "marketPrice", state, symbol)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+marketDataColumns,
		md.ID, md.DataSource, md.Date, md.MarketPrice, md.State, md.Symbol)
	if err != nil {
		return model.MarketData{}, fmt.Errorf("insert market data: %w", err)
	}
	return item, nil
}
