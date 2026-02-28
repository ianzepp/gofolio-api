package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type PropertyRepository struct {
	db db.DBTX
}

func NewPropertyRepository(db db.DBTX) *PropertyRepository {
	return &PropertyRepository{db: db}
}

func (r *PropertyRepository) FindByKey(ctx context.Context, key string) (model.Property, error) {
	rows, err := r.db.Query(ctx,
		`SELECT key, value FROM "Property" WHERE key = $1`, key)
	if err != nil {
		return model.Property{}, fmt.Errorf("query property: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Property])
}

func (r *PropertyRepository) Upsert(ctx context.Context, p model.Property) (model.Property, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "Property" (key, value)
		 VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = $2
		 RETURNING key, value`,
		p.Key, p.Value)
	if err != nil {
		return model.Property{}, fmt.Errorf("upsert property: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Property])
}

func (r *PropertyRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "Property" WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("delete property: %w", err)
	}
	return nil
}
