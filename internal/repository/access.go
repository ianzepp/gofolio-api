// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// AccessRepository handles persistence for the Access entity, which models
// portfolio-sharing grants from one Ghostfolio user (the owner, "userId") to
// another ("granteeUserId"). An Access row controls what a grantee can see or
// do within the owner's portfolio — permissions and per-grant settings are
// stored as Postgres arrays and JSONB respectively.
type AccessRepository struct {
	db db.DBTX
}

// NewAccessRepository constructs an AccessRepository backed by the provided
// database handle.
func NewAccessRepository(db db.DBTX) *AccessRepository {
	return &AccessRepository{db: db}
}

// accessColumns is the canonical SELECT list for Access rows.
const accessColumns = `id, alias, "createdAt", "granteeUserId",
	permissions, settings, "updatedAt", "userId"`

// FindByUserID returns all access grants created by userID (i.e. grants where
// the calling user is the portfolio owner), ordered newest first.
func (r *AccessRepository) FindByUserID(ctx context.Context, userID string) ([]model.Access, error) {
	var items []model.Access
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+accessColumns+`
		 FROM "Access"
		 WHERE "userId" = $1
		 ORDER BY "createdAt" DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accesses: %w", err)
	}
	return items, nil
}

// FindByID fetches a single access grant by its primary key without an
// ownership restriction. Use FindByIDAndUserID for user-scoped lookups.
func (r *AccessRepository) FindByID(ctx context.Context, id string) (model.Access, error) {
	var item model.Access
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+accessColumns+`
		 FROM "Access"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Access{}, fmt.Errorf("query access: %w", err)
	}
	return item, nil
}

// FindByIDAndUserID fetches a grant scoped to the owning user, acting as a
// combined lookup-and-ownership-check. Returns ErrNoRows when the grant does
// not exist or belongs to a different user.
func (r *AccessRepository) FindByIDAndUserID(ctx context.Context, id string, userID string) (model.Access, error) {
	var item model.Access
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+accessColumns+`
		 FROM "Access"
		 WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return model.Access{}, fmt.Errorf("query access by user: %w", err)
	}
	return item, nil
}

// Create inserts a new access grant and returns the full persisted record.
// The caller must supply a pre-generated ID and populate all required fields.
func (r *AccessRepository) Create(ctx context.Context, a model.Access) (model.Access, error) {
	var item model.Access
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "Access" (id, alias, "granteeUserId", permissions,
		                       settings, "userId")
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+accessColumns,
		a.ID, a.Alias, a.GranteeUserId, a.Permissions,
		a.Settings, a.UserId)
	if err != nil {
		return model.Access{}, fmt.Errorf("insert access: %w", err)
	}
	return item, nil
}

// Update replaces the mutable fields of an access grant (alias, permissions,
// settings) without verifying ownership. Intended for admin-level paths.
// NOTE: alias is a pointer to distinguish an explicit null from an absent value.
func (r *AccessRepository) Update(ctx context.Context, id string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	var item model.Access
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`UPDATE "Access"
		 SET alias = $2, permissions = $3, settings = $4
		 WHERE id = $1
		 RETURNING `+accessColumns,
		id, alias, permissions, settings)
	if err != nil {
		return model.Access{}, fmt.Errorf("update access: %w", err)
	}
	return item, nil
}

// UpdateByUserID updates a grant's mutable fields only when it is owned by
// userID, enforcing ownership at the query level.
func (r *AccessRepository) UpdateByUserID(ctx context.Context, id string, userID string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	var item model.Access
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`UPDATE "Access"
		 SET alias = $3, permissions = $4, settings = $5
		 WHERE id = $1 AND "userId" = $2
		 RETURNING `+accessColumns,
		id, userID, alias, permissions, settings)
	if err != nil {
		return model.Access{}, fmt.Errorf("update access by user: %w", err)
	}
	return item, nil
}

// Delete removes an access grant by ID without ownership verification.
// Use DeleteByUserID for user-scoped deletion.
func (r *AccessRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM "Access" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete access: %w", err)
	}
	return nil
}

// DeleteByUserID removes the grant identified by id only when it belongs to
// userID. Returns the count of affected rows so callers can distinguish
// not-found (0) from success (1) without a separate lookup.
func (r *AccessRepository) DeleteByUserID(ctx context.Context, id string, userID string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM "Access" WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return 0, fmt.Errorf("delete access by user: %w", err)
	}
	return result.RowsAffected()
}
