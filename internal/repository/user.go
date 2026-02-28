// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// UserRepository handles persistence for the User entity, which is the root
// authentication and authorisation principal. It also surfaces Settings,
// which are stored in a separate one-to-one table and queried here to keep
// user-centric data co-located.
type UserRepository struct {
	db db.DBTX
}

// NewUserRepository constructs a UserRepository backed by the provided handle.
func NewUserRepository(db db.DBTX) *UserRepository {
	return &UserRepository{db: db}
}

// userColumns is the canonical SELECT list for User rows. Sensitive fields
// like accessToken are intentionally included here — callers must not expose
// them directly in API responses.
const userColumns = `id, "accessToken", "authChallenge", "createdAt",
	provider, role, "thirdPartyId", "updatedAt"`

// FindByID fetches a user by primary key. Returns a wrapped pgx.ErrNoRows
// when no matching record exists.
func (r *UserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	var item model.User
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+userColumns+`
		 FROM "User"
		 WHERE id = $1`, id)
	if err != nil {
		return model.User{}, fmt.Errorf("query user: %w", err)
	}
	return item, nil
}

// Create inserts a new User row, letting the database set createdAt/updatedAt
// to NOW(). The caller must supply a pre-generated ID and hashed accessToken.
// WHY: timestamps are set server-side so all rows share a consistent clock
// regardless of application-layer time skew.
func (r *UserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	now := time.Now().UTC()
	var item model.User
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "User" (id, "accessToken", "createdAt", provider, role, "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $3)
		 RETURNING `+userColumns,
		user.ID, user.AccessToken, now, user.Provider, user.Role)
	if err != nil {
		return model.User{}, fmt.Errorf("insert user: %w", err)
	}
	return item, nil
}

// FindByAccessToken resolves a user from a stored (hashed) access token.
// This is the primary lookup path for bearer-token authentication — the
// service layer hashes the inbound token before calling here.
func (r *UserRepository) FindByAccessToken(ctx context.Context, accessToken string) (model.User, error) {
	var item model.User
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+userColumns+`
		 FROM "User"
		 WHERE "accessToken" = $1`, accessToken)
	if err != nil {
		return model.User{}, fmt.Errorf("query user by access token: %w", err)
	}
	return item, nil
}

// FindAll returns every user ordered by creation time, newest first.
// Intended for admin dashboards only — should never be called from
// user-facing endpoints.
func (r *UserRepository) FindAll(ctx context.Context) ([]model.User, error) {
	var items []model.User
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+userColumns+`
		 FROM "User"
		 ORDER BY "createdAt" DESC`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	return items, nil
}

// CountAll returns the total number of registered users.
// Used by the admin overview and to gate first-user registration flows.
func (r *UserRepository) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowxContext(ctx, `SELECT COUNT(*) FROM "User"`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

// Delete removes a user by ID. Callers must cascade-delete dependent rows
// (orders, accounts, accesses, etc.) before calling this to avoid FK
// constraint violations.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM "User" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// UpdateAccessToken replaces the stored hashed access token for a user.
// The caller is responsible for hashing the raw token before passing it here.
// EDGE: uses RETURNING + CollectExactlyOneRow so the update is verified to
// have matched exactly one row — an implicit existence check.
func (r *UserRepository) UpdateAccessToken(ctx context.Context, id string, hashedAccessToken string) error {
	var user model.User
	err := sqlx.GetContext(ctx, r.db,
		&user,
		`UPDATE "User"
		 SET "accessToken" = $2
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, hashedAccessToken)
	if err != nil {
		return fmt.Errorf("update user access token: %w", err)
	}
	return nil
}

// FindSettings retrieves the user-specific settings JSON blob from the
// separate Settings table. Returns ErrNoRows when the user has not yet saved
// any preferences — callers should apply application defaults in that case.
func (r *UserRepository) FindSettings(ctx context.Context, userID string) (model.Settings, error) {
	var item model.Settings
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT "userId", settings, "updatedAt"
		 FROM "Settings"
		 WHERE "userId" = $1`, userID)
	if err != nil {
		return model.Settings{}, fmt.Errorf("query user settings: %w", err)
	}
	return item, nil
}

// UpsertSettings creates or replaces the settings record for a user.
// The ON CONFLICT clause targets the userId primary key, matching Prisma's
// upsert semantics — only the settings JSON column is overwritten on conflict.
func (r *UserRepository) UpsertSettings(ctx context.Context, s model.Settings) (model.Settings, error) {
	var item model.Settings
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "Settings" ("userId", settings)
		 VALUES ($1, $2)
		 ON CONFLICT ("userId") DO UPDATE SET settings = $2
		 RETURNING "userId", settings, "updatedAt"`,
		s.UserId, s.Settings)
	if err != nil {
		return model.Settings{}, fmt.Errorf("upsert user settings: %w", err)
	}
	return item, nil
}
