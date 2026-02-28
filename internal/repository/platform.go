package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type PlatformRepository struct {
	db db.DBTX
}

func NewPlatformRepository(db db.DBTX) *PlatformRepository {
	return &PlatformRepository{db: db}
}

func (r *PlatformRepository) FindAll(ctx context.Context) ([]model.Platform, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, url
		 FROM "Platform"
		 ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query platforms: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Platform])
}

func (r *PlatformRepository) FindByID(ctx context.Context, id string) (model.Platform, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, url
		 FROM "Platform"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Platform{}, fmt.Errorf("query platform: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Platform])
}

func (r *PlatformRepository) Create(ctx context.Context, p model.Platform) (model.Platform, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "Platform" (id, name, url)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, url`,
		p.ID, p.Name, p.Url)
	if err != nil {
		return model.Platform{}, fmt.Errorf("insert platform: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Platform])
}

func (r *PlatformRepository) Update(ctx context.Context, p model.Platform) (model.Platform, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE "Platform" SET name = $2, url = $3
		 WHERE id = $1
		 RETURNING id, name, url`,
		p.ID, p.Name, p.Url)
	if err != nil {
		return model.Platform{}, fmt.Errorf("update platform: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Platform])
}

func (r *PlatformRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "Platform" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete platform: %w", err)
	}
	return nil
}
