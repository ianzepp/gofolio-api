package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type AccountBalanceRepository struct {
	db db.DBTX
}

func NewAccountBalanceRepository(db db.DBTX) *AccountBalanceRepository {
	return &AccountBalanceRepository{db: db}
}

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
		`INSERT INTO "AccountBalance" (id, "accountId", "userId", date, value)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT ("accountId", date) DO UPDATE
		   SET value = EXCLUDED.value, "updatedAt" = now()
		 RETURNING id, "accountId", "createdAt", date, "updatedAt", "userId", value`,
		ab.ID, ab.AccountId, ab.UserId, ab.Date, ab.Value)
	if err != nil {
		return model.AccountBalance{}, fmt.Errorf("upsert account-balance: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.AccountBalance])
}

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

// ResetHours truncates a time value to midnight UTC, matching NestJS resetHours behavior.
func ResetHours(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
