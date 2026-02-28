// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// SymbolProfileRepository handles persistence for SymbolProfile, which is the
// canonical metadata record for a tradeable instrument (stock, ETF, crypto,
// etc.). Each profile is uniquely identified by (dataSource, symbol) and
// carries rich instrument metadata — asset class, sectors, countries, FIGI
// identifiers, and a scraper configuration used by the data-collection layer.
//
// Orders reference SymbolProfiles via symbolProfileId, so a profile must exist
// before an order for that instrument can be recorded.
type SymbolProfileRepository struct {
	db db.DBTX
}

// NewSymbolProfileRepository constructs a SymbolProfileRepository backed by
// the provided database handle.
func NewSymbolProfileRepository(db db.DBTX) *SymbolProfileRepository {
	return &SymbolProfileRepository{db: db}
}

// symbolProfileColumns is the canonical SELECT list for SymbolProfile rows.
// The column count is large because SymbolProfile aggregates all known
// identifiers (ISIN, CUSIP, FIGI, etc.) into a single table to avoid joins
// during price-calculation hot paths.
const symbolProfileColumns = `id, "assetClass", "assetSubClass", comment,
	COALESCE(countries, '[]'::jsonb) AS countries, "createdAt", currency, cusip, "dataSource", figi,
	"figiComposite", "figiShareClass", COALESCE(holdings, '[]'::jsonb) AS holdings, "isActive", isin,
	name, COALESCE("scraperConfiguration", '{}'::jsonb) AS "scraperConfiguration", COALESCE(sectors, '[]'::jsonb) AS sectors, symbol, COALESCE("symbolMapping", '{}'::jsonb) AS "symbolMapping",
	"updatedAt", url, "userId"`

// FindByID retrieves a symbol profile by its primary key. Returns a wrapped
// ErrNoRows when no matching record exists.
func (r *SymbolProfileRepository) FindByID(ctx context.Context, id string) (model.SymbolProfile, error) {
	var item model.SymbolProfile
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 WHERE id = $1`, id)
	if err != nil {
		return model.SymbolProfile{}, fmt.Errorf("query symbol profile: %w", err)
	}
	return item, nil
}

// FindByDataSourceAndSymbol looks up a profile by its natural composite key.
// This is the lookup used when importing orders — the caller resolves the
// ticker+source pair to a stable UUID before inserting the order row.
func (r *SymbolProfileRepository) FindByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (model.SymbolProfile, error) {
	var item model.SymbolProfile
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 WHERE "dataSource" = $1 AND symbol = $2`, dataSource, symbol)
	if err != nil {
		return model.SymbolProfile{}, fmt.Errorf("query symbol profile by source/symbol: %w", err)
	}
	return item, nil
}

// CountByDataSource returns the number of profiles registered under a given
// data source. Used to surface per-provider coverage metrics in admin views.
func (r *SymbolProfileRepository) CountByDataSource(ctx context.Context, dataSource string) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx,
		`SELECT COUNT(*) FROM "SymbolProfile" WHERE "dataSource" = $1`, dataSource).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count symbol profiles by data source: %w", err)
	}
	return count, nil
}

// FindByIDs fetches multiple profiles in a single query using a Postgres
// ANY($1) array parameter. Returns an empty slice immediately when ids is
// empty to avoid a no-op database round-trip.
// WHY: batch lookup avoids N+1 queries when hydrating a list of orders with
// their associated instrument metadata.
func (r *SymbolProfileRepository) FindByIDs(ctx context.Context, ids []string) ([]model.SymbolProfile, error) {
	if len(ids) == 0 {
		return []model.SymbolProfile{}, nil
	}
	query, args, err := sqlx.In(
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 WHERE id IN (?)
		 ORDER BY name ASC`, ids)
	if err != nil {
		return nil, fmt.Errorf("build symbol profiles query: %w", err)
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)
	var items []model.SymbolProfile
	err = sqlx.SelectContext(ctx, r.db, &items, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query symbol profiles by ids: %w", err)
	}
	return items, nil
}

// SearchByQuery performs a case-insensitive substring search on name and
// symbol within a given data source, returning up to 50 results. The hard
// limit prevents accidental full-table scans on broad search terms.
// TRADE-OFF: ILIKE with a leading wildcard prevents index use on name/symbol
// but is acceptable given the 50-row cap and the expectation that the symbol
// profile table is small relative to market data or orders.
func (r *SymbolProfileRepository) SearchByQuery(ctx context.Context, dataSource string, query string) ([]model.SymbolProfile, error) {
	pattern := "%" + query + "%"
	var items []model.SymbolProfile
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 WHERE "dataSource" = $1
		   AND (LOWER(name) LIKE LOWER($2) OR LOWER(symbol) LIKE LOWER($2))
		 ORDER BY symbol ASC
		 LIMIT 50`, dataSource, pattern)
	if err != nil {
		return nil, fmt.Errorf("search symbol profiles: %w", err)
	}
	return items, nil
}

// FindAll retrieves every symbol profile ordered by ticker symbol.
// Used by export and admin endpoints; avoid calling from performance-sensitive
// paths as the result set can grow large over time.
func (r *SymbolProfileRepository) FindAll(ctx context.Context) ([]model.SymbolProfile, error) {
	var items []model.SymbolProfile
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 ORDER BY symbol ASC`)
	if err != nil {
		return nil, fmt.Errorf("query symbol profiles: %w", err)
	}
	return items, nil
}
