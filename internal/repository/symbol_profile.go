package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type SymbolProfileRepository struct {
	db db.DBTX
}

func NewSymbolProfileRepository(db db.DBTX) *SymbolProfileRepository {
	return &SymbolProfileRepository{db: db}
}

const symbolProfileColumns = `id, "assetClass", "assetSubClass", comment,
	countries, "createdAt", currency, cusip, "dataSource", figi,
	"figiComposite", "figiShareClass", holdings, "isActive", isin,
	name, "scraperConfiguration", sectors, symbol, "symbolMapping",
	"updatedAt", url, "userId"`

func (r *SymbolProfileRepository) FindByID(ctx context.Context, id string) (model.SymbolProfile, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 WHERE id = $1`, id)
	if err != nil {
		return model.SymbolProfile{}, fmt.Errorf("query symbol profile: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.SymbolProfile])
}

func (r *SymbolProfileRepository) FindByDataSourceAndSymbol(ctx context.Context, dataSource string, symbol string) (model.SymbolProfile, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 WHERE "dataSource" = $1 AND symbol = $2`, dataSource, symbol)
	if err != nil {
		return model.SymbolProfile{}, fmt.Errorf("query symbol profile by source/symbol: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.SymbolProfile])
}

func (r *SymbolProfileRepository) FindAll(ctx context.Context) ([]model.SymbolProfile, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+symbolProfileColumns+`
		 FROM "SymbolProfile"
		 ORDER BY symbol ASC`)
	if err != nil {
		return nil, fmt.Errorf("query symbol profiles: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.SymbolProfile])
}
