// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

// PlatformRepository handles persistence for Platform records, which are
// administrator-managed references to brokerage or exchange platforms
// (e.g. "Interactive Brokers", "Coinbase"). Platforms are optional metadata
// on accounts — an account may omit a platform without any functional impact
// on portfolio calculations.
//
// Platforms are global (not user-scoped), so all CRUD operations here omit
// ownership checks. Only administrators should invoke write methods.
type PlatformRepository struct {
	db db.DBTX
}

// NewPlatformRepository constructs a PlatformRepository backed by the
// provided database handle.
func NewPlatformRepository(db db.DBTX) *PlatformRepository {
	return &PlatformRepository{db: db}
}

// FindAll returns every platform ordered by name. The result is used to
// populate dropdowns in the account editor UI.
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

// FindByID fetches a platform by its primary key. Returns ErrNoRows when no
// matching record exists.
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

// Create inserts a new platform record and returns the persisted row.
// The caller must supply a pre-generated ID.
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

// Update replaces the name and URL of an existing platform. Returns ErrNoRows
// when no matching record exists.
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

// Delete removes a platform by ID. The caller should verify that no accounts
// still reference this platform (platformId FK) before deletion; the database
// will reject the operation with a constraint error if accounts remain.
func (r *PlatformRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "Platform" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete platform: %w", err)
	}
	return nil
}
