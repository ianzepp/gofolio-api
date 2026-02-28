package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type AccountRepository struct {
	db db.DBTX
}

func NewAccountRepository(db db.DBTX) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) FindByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, "userId", balance, comment, "createdAt", currency,
		        "isExcluded", name, "platformId", "updatedAt"
		 FROM "Account"
		 WHERE "userId" = $1
		 ORDER BY name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Account])
}

func (r *AccountRepository) FindByID(ctx context.Context, id string, userID string) (model.Account, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, "userId", balance, comment, "createdAt", currency,
		        "isExcluded", name, "platformId", "updatedAt"
		 FROM "Account"
		 WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return model.Account{}, fmt.Errorf("query account: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Account])
}

func (r *AccountRepository) Create(ctx context.Context, acct model.Account) (model.Account, error) {
	rows, err := r.db.Query(ctx,
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
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Account])
}

func (r *AccountRepository) Update(ctx context.Context, acct model.Account) (model.Account, error) {
	rows, err := r.db.Query(ctx,
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
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Account])
}

func (r *AccountRepository) Delete(ctx context.Context, id string, userID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM "Account" WHERE id = $1 AND "userId" = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}

func (r *AccountRepository) UpdateBalance(ctx context.Context, id string, userID string, delta float64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "Account" SET balance = balance + $3
		 WHERE id = $1 AND "userId" = $2`, id, userID, delta)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}
	return nil
}
