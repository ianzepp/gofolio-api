package service

import (
	"context"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// AccountBalanceService manages time-series cash balance snapshots for a
// brokerage account. Unlike the Account.Balance field (which holds the current
// balance), AccountBalance rows record the balance at a specific point in time,
// enabling the portfolio engine to plot net worth over time accurately.
type AccountBalanceService struct {
	db *sqlx.DB
}

// NewAccountBalanceService constructs an AccountBalanceService backed by the
// given pool.
func NewAccountBalanceService(db *sqlx.DB) *AccountBalanceService {
	return &AccountBalanceService{db: db}
}

// GetAccountBalance retrieves a single balance record by ID, scoped to the
// given user to prevent cross-user data access.
func (s *AccountBalanceService) GetAccountBalance(ctx context.Context, id string, userID string) (model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.db)
	return repo.FindByID(ctx, id, userID)
}

func (s *AccountBalanceService) GetAccountBalances(ctx context.Context, accountID string, userID string) ([]model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.db)
	return repo.FindByAccountID(ctx, accountID, userID)
}

// CreateOrUpdateAccountBalance upserts a balance snapshot. The upsert
// semantics (keyed on account + date) mean that recording the same date twice
// updates the existing row rather than creating a duplicate, which is the
// correct behaviour for balance corrections.
func (s *AccountBalanceService) CreateOrUpdateAccountBalance(ctx context.Context, ab model.AccountBalance) (model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.db)
	return repo.Upsert(ctx, ab)
}

func (s *AccountBalanceService) DeleteAccountBalance(ctx context.Context, id string, userID string) (model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.db)
	return repo.Delete(ctx, id, userID)
}
