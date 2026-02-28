// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

// AccountBalanceRepository handles persistence for point-in-time cash balance
// snapshots attached to a brokerage account. Unlike the Account.balance field
// (which is a running total), AccountBalance stores a historical series of
// dated balance observations — typically one per day — so the portfolio
// timeline can reflect actual cash positions over time.
//
// The unique constraint on (accountId, date) means only one snapshot per
// account per calendar day is allowed; writes use upsert semantics to support
// idempotent imports.
type AccountBalanceRepository struct {
	db db.DBTX
}

// NewAccountBalanceRepository constructs an AccountBalanceRepository backed
// by the provided database handle.
func NewAccountBalanceRepository(db db.DBTX) *AccountBalanceRepository {
	return &AccountBalanceRepository{db: db}
}

// FindByID fetches a single balance snapshot by primary key, scoped to
// userID. Returns ErrNoRows when not found or ownership does not match.
func (r *AccountBalanceRepository) FindByID(ctx context.Context, id string, userID string) (model.AccountBalance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, "accountId", "createdAt", date, "updatedAt", "userId", value
		 FROM "AccountBalance"
		 WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return model.AccountBalance{}, fmt.Errorf("query account-balance: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.AccountBalance])
}

// FindByAccountID returns all balance snapshots for a given account, ordered
// chronologically ascending. The userId guard prevents cross-user data leaks
// even when the accountId is known to an attacker.
func (r *AccountBalanceRepository) FindByAccountID(ctx context.Context, accountID string, userID string) ([]model.AccountBalance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, "accountId", "createdAt", date, "updatedAt", "userId", value
		 FROM "AccountBalance"
		 WHERE "accountId" = $1 AND "userId" = $2
		 ORDER BY date ASC`, accountID, userID)
	if err != nil {
		return nil, fmt.Errorf("query account-balances: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.AccountBalance])
}

// Upsert creates or updates an AccountBalance keyed on (accountId, date).
// Matches NestJS Prisma upsert semantics on the unique constraint.
func (r *AccountBalanceRepository) Upsert(ctx context.Context, ab model.AccountBalance) (model.AccountBalance, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "AccountBalance" (id, "accountId", "createdAt", date, "updatedAt", "userId", value)
		 VALUES ($1, $2, NOW(), $4, NOW(), $3, $5)
		 ON CONFLICT ("accountId", date) DO UPDATE
		   SET value = EXCLUDED.value, "updatedAt" = now()
		 RETURNING id, "accountId", "createdAt", date, "updatedAt", "userId", value`,
		ab.ID, ab.AccountId, ab.UserId, ab.Date, ab.Value)
	if err != nil {
		return model.AccountBalance{}, fmt.Errorf("upsert account-balance: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.AccountBalance])
}

// Delete removes a balance snapshot by ID, scoped to userID, and returns the
// deleted record. Returning the deleted row avoids a subsequent SELECT to
// confirm what was removed — useful for audit logging in the service layer.
func (r *AccountBalanceRepository) Delete(ctx context.Context, id string, userID string) (model.AccountBalance, error) {
	rows, err := r.db.Query(ctx,
		`DELETE FROM "AccountBalance"
		 WHERE id = $1 AND "userId" = $2
		 RETURNING id, "accountId", "createdAt", date, "updatedAt", "userId", value`, id, userID)
	if err != nil {
		return model.AccountBalance{}, fmt.Errorf("delete account-balance: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.AccountBalance])
}

// ResetHours truncates a time value to midnight UTC, replicating the NestJS
// resetHours() helper used throughout the original codebase. It is placed here
// because AccountBalance dates must always be day-granular — storing intra-day
// timestamps would violate the (accountId, date) unique constraint.
func ResetHours(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
