// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// PropertyRepository handles persistence for the Property key-value store,
// which is a simple string→string table used as a system-wide configuration
// registry. Ghostfolio stores feature flags, last-fetch timestamps, and other
// runtime state here rather than in environment variables so that they can be
// updated at runtime without restarting the server.
//
// Values are stored as text; the service layer is responsible for any type
// coercion (e.g. parsing booleans or timestamps from the stored string).
type PropertyRepository struct {
	db db.DBTX
}

// NewPropertyRepository constructs a PropertyRepository backed by the provided
// database handle.
func NewPropertyRepository(db db.DBTX) *PropertyRepository {
	return &PropertyRepository{db: db}
}

// FindByKey retrieves a single property by its string key. Returns a wrapped
// ErrNoRows when the key does not exist — callers should apply a default value
// in that case rather than treating it as a hard error.
func (r *PropertyRepository) FindByKey(ctx context.Context, key string) (model.Property, error) {
	var item model.Property
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT key, value FROM "Property" WHERE key = $1`, key)
	if err != nil {
		return model.Property{}, fmt.Errorf("query property: %w", err)
	}
	return item, nil
}

// FindAll retrieves every property ordered by key. Used by admin endpoints
// to display and bulk-edit system configuration.
func (r *PropertyRepository) FindAll(ctx context.Context) ([]model.Property, error) {
	var items []model.Property
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT key, value FROM "Property" ORDER BY key ASC`)
	if err != nil {
		return nil, fmt.Errorf("query properties: %w", err)
	}
	return items, nil
}

// Upsert creates or overwrites a property by key. The ON CONFLICT clause
// targets the primary key so re-writing the same value is idempotent.
// WHY: upsert rather than separate insert/update simplifies the service layer,
// which never needs to pre-check existence before writing a config value.
func (r *PropertyRepository) Upsert(ctx context.Context, p model.Property) (model.Property, error) {
	var item model.Property
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "Property" (key, value)
		 VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = $2
		 RETURNING key, value`,
		p.Key, p.Value)
	if err != nil {
		return model.Property{}, fmt.Errorf("upsert property: %w", err)
	}
	return item, nil
}

// Delete removes a property by key. Deleting a non-existent key is a no-op
// (the Exec succeeds with zero rows affected).
func (r *PropertyRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM "Property" WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("delete property: %w", err)
	}
	return nil
}
