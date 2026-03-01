// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ianzepp/gofolio-api/internal/db"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/jmoiron/sqlx"
)

// AccountRepository handles persistence for broker/bank account records.
// An Account groups orders by custodian and carries a running cash balance.
// Each account belongs to a single User and optionally references a Platform
// (e.g. "Interactive Brokers") for display purposes.
//
// NOTE: all account lookups include "userId" in the WHERE clause so that
// ownership is enforced at the query level rather than in application code.
type AccountRepository struct {
	db db.DBTX
}

// NewAccountRepository constructs an AccountRepository backed by the provided
// database handle.
func NewAccountRepository(db db.DBTX) *AccountRepository {
	return &AccountRepository{db: db}
}

// FindByUserID returns all accounts owned by userID, sorted alphabetically
// by account name.
func (r *AccountRepository) FindByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	var items []model.Account
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT id, "userId", balance, comment, "createdAt", currency,
		        "isExcluded", name, "platformId", "updatedAt"
		 FROM "Account"
		 WHERE "userId" = $1
		 ORDER BY name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	return items, nil
}

// FindByID fetches a single account by its primary key, scoped to userID.
// Returns ErrNoRows when the account does not exist or belongs to another user.
func (r *AccountRepository) FindByID(ctx context.Context, id string, userID string) (model.Account, error) {
	var item model.Account
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT id, "userId", balance, comment, "createdAt", currency,
		        "isExcluded", name, "platformId", "updatedAt"
		 FROM "Account"
		 WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return model.Account{}, fmt.Errorf("query account: %w", err)
	}
	return item, nil
}

// Create inserts a new account row and returns the full persisted record
// including database-generated timestamps. The caller must supply a
// pre-generated ID.
func (r *AccountRepository) Create(ctx context.Context, acct model.Account) (model.Account, error) {
	var item model.Account
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "Account" (id, "userId", balance, comment, currency,
		                        "isExcluded", name, "platformId")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, "userId", balance, comment, "createdAt", currency,
		           "isExcluded", name, "platformId", "updatedAt"`,
		acct.ID, acct.UserId, acct.Balance, acct.Comment, acct.Currency,
		acct.IsExcluded, acct.Name, acct.PlatformId)
	if err != nil {
		return model.Account{}, fmt.Errorf("insert account: %w", err)
	}
	return item, nil
}

// Update replaces all mutable fields on an account and enforces ownership via
// the WHERE clause. Returns ErrNoRows when the account does not exist or
// belongs to a different user.
func (r *AccountRepository) Update(ctx context.Context, acct model.Account) (model.Account, error) {
	var item model.Account
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`UPDATE "Account"
		 SET balance = $3, comment = $4, currency = $5,
		     "isExcluded" = $6, name = $7, "platformId" = $8
		 WHERE id = $1 AND "userId" = $2
		 RETURNING id, "userId", balance, comment, "createdAt", currency,
		           "isExcluded", name, "platformId", "updatedAt"`,
		acct.ID, acct.UserId, acct.Balance, acct.Comment, acct.Currency,
		acct.IsExcluded, acct.Name, acct.PlatformId)
	if err != nil {
		return model.Account{}, fmt.Errorf("update account: %w", err)
	}
	return item, nil
}

// Delete removes an account scoped to userID. The caller must ensure all
// associated orders and account balances have been removed first to avoid
// FK constraint violations.
func (r *AccountRepository) Delete(ctx context.Context, id string, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM "Account" WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

// UpdateBalance applies a signed delta to the account's running cash balance
// atomically using SQL arithmetic (balance = balance + delta). This avoids
// a read-modify-write cycle and is safe under concurrent updates.
// WHY: a delta-based update rather than a set-based update prevents the race
// condition where two concurrent writers overwrite each other's changes.
func (r *AccountRepository) UpdateBalance(ctx context.Context, id string, userID string, delta float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE "Account" SET balance = balance + $3
		 WHERE id = $1 AND "userId" = $2`, id, userID, delta)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}
	return nil
}
