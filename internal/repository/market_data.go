package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type MarketDataRepository struct {
	db db.DBTX
}

func NewMarketDataRepository(db db.DBTX) *MarketDataRepository {
	return &MarketDataRepository{db: db}
}

const marketDataColumns = `id, "createdAt", "dataSource", date,
	"marketPrice", state, symbol`

func (r *MarketDataRepository) FindBySymbol(ctx context.Context, dataSource string, symbol string) ([]model.MarketData, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2
		 ORDER BY date DESC`, dataSource, symbol)
	if err != nil {
		return nil, fmt.Errorf("query market data: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.MarketData])
}

func (r *MarketDataRepository) FindBySymbolAndDateRange(ctx context.Context, dataSource string, symbol string, from time.Time, to time.Time) ([]model.MarketData, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2
		   AND date >= $3 AND date <= $4
		 ORDER BY date ASC`, dataSource, symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("query market data range: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.MarketData])
}

func (r *MarketDataRepository) CountByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2`, dataSource, symbol).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count market data: %w", err)
	}
	return count, nil
}

func (r *MarketDataRepository) FindLatestByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (model.MarketData, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2
		 ORDER BY date DESC
		 LIMIT 1`, dataSource, symbol)
	if err != nil {
		return model.MarketData{}, fmt.Errorf("query latest market data: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.MarketData])
}

func (r *MarketDataRepository) FindLatestBySymbols(ctx context.Context, dataSource string, symbols []string) ([]model.MarketData, error) {
	if len(symbols) == 0 {
		return []model.MarketData{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (symbol) `+marketDataColumns+`
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = ANY($2)
		 ORDER BY symbol, date DESC`, dataSource, symbols)
	if err != nil {
		return nil, fmt.Errorf("query latest market data by symbols: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.MarketData])
}

func (r *MarketDataRepository) UpsertMarketPrice(ctx context.Context, dataSource string, symbol string, date time.Time, price float64, state model.MarketDataState) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO "MarketData" (id, "dataSource", date, "marketPrice", state, symbol)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5)
		 ON CONFLICT ("dataSource", date, symbol) DO UPDATE
		   SET "marketPrice" = EXCLUDED."marketPrice",
		       state = EXCLUDED.state`,
		dataSource, date, price, state, symbol)
	if err != nil {
		return fmt.Errorf("upsert market price: %w", err)
	}
	return nil
}

func (r *MarketDataRepository) Create(ctx context.Context, md model.MarketData) (model.MarketData, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "MarketData" (id, "dataSource", date, "marketPrice", state, symbol)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+marketDataColumns,
		md.ID, md.DataSource, md.Date, md.MarketPrice, md.State, md.Symbol)
	if err != nil {
		return model.MarketData{}, fmt.Errorf("insert market data: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.MarketData])
}
