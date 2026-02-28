package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type AccessRepository struct {
	db db.DBTX
}

func NewAccessRepository(db db.DBTX) *AccessRepository {
	return &AccessRepository{db: db}
}

const accessColumns = `id, alias, "createdAt", "granteeUserId",
	permissions, settings, "updatedAt", "userId"`

func (r *AccessRepository) FindByUserID(ctx context.Context, userID string) ([]model.Access, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+accessColumns+`
		 FROM "Access"
		 WHERE "userId" = $1
		 ORDER BY "createdAt" DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accesses: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Access])
}

func (r *AccessRepository) FindByID(ctx context.Context, id string) (model.Access, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+accessColumns+`
		 FROM "Access"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Access{}, fmt.Errorf("query access: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Access])
}

func (r *AccessRepository) Create(ctx context.Context, a model.Access) (model.Access, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "Access" (id, alias, "granteeUserId", permissions,
		                       settings, "userId")
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+accessColumns,
		a.ID, a.Alias, a.GranteeUserId, a.Permissions,
		a.Settings, a.UserId)
	if err != nil {
		return model.Access{}, fmt.Errorf("insert access: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Access])
}

func (r *AccessRepository) Update(ctx context.Context, id string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE "Access"
		 SET alias = $2, permissions = $3, settings = $4
		 WHERE id = $1
		 RETURNING `+accessColumns,
		id, alias, permissions, settings)
	if err != nil {
		return model.Access{}, fmt.Errorf("update access: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Access])
}

func (r *AccessRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "Access" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete access: %w", err)
	}
	return nil
}
