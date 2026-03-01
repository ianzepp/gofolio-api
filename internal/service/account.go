package service

import (
	"context"
	"fmt"

	"github.com/ianzepp/gofolio-api/internal/db"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// AccountService manages brokerage and cash accounts owned by users. Accounts
// are the top-level container for activities (orders) and balances, and they
// carry a currency so that the portfolio engine can convert positions to the
// user's base currency.
type AccountService struct {
	db *sqlx.DB
}

// NewAccountService constructs an AccountService backed by the given database.
func NewAccountService(db *sqlx.DB) *AccountService {
	return &AccountService{db: db}
}

// GetAccounts returns all accounts belonging to the given user.
func (s *AccountService) GetAccounts(ctx context.Context, userID string) ([]model.Account, error) {
	repo := repository.NewAccountRepository(s.db)
	return repo.FindByUserID(ctx, userID)
}

func (s *AccountService) GetAccount(ctx context.Context, id string, userID string) (model.Account, error) {
	repo := repository.NewAccountRepository(s.db)
	return repo.FindByID(ctx, id, userID)
}

func (s *AccountService) CreateAccount(ctx context.Context, acct model.Account) (model.Account, error) {
	repo := repository.NewAccountRepository(s.db)
	return repo.Create(ctx, acct)
}

func (s *AccountService) UpdateAccount(ctx context.Context, acct model.Account) (model.Account, error) {
	repo := repository.NewAccountRepository(s.db)
	return repo.Update(ctx, acct)
}

func (s *AccountService) DeleteAccount(ctx context.Context, id string, userID string) error {
	repo := repository.NewAccountRepository(s.db)
	return repo.Delete(ctx, id, userID)
}

// TransferRequest carries the parameters for an atomic balance transfer between
// two accounts. Both accounts must be owned by authenticated users, enforced
// via the FromUserID/ToUserID fields that are threaded through to the
// repository UPDATE WHERE clause.
type TransferRequest struct {
	FromID     string
	FromUserID string
	ToID       string
	ToUserID   string
	Amount     float64
}

// TransferBalance moves Amount from one account to another atomically within a
// single database transaction. The debit and credit are executed sequentially
// inside the transaction so that a partial failure leaves no net balance change.
//
// WHY: Using db.WithTx ensures that the two UPDATE statements share a
// transaction even though the repository is constructed from a pgx.Tx rather
// than the pool, preventing a race condition where the debit succeeds but the
// credit fails.
func (s *AccountService) TransferBalance(ctx context.Context, req TransferRequest) error {
	return db.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		repo := repository.NewAccountRepository(tx)
		// Lock the source account row and verify sufficient balance inside
		// the transaction to prevent concurrent overdraw.
		balance, err := repo.GetBalanceForUpdate(ctx, req.FromID, req.FromUserID)
		if err != nil {
			return fmt.Errorf("lock source account: %w", err)
		}
		if balance < req.Amount {
			return fmt.Errorf("insufficient balance")
		}
		if err := repo.UpdateBalance(ctx, req.FromID, req.FromUserID, -req.Amount); err != nil {
			return err
		}
		return repo.UpdateBalance(ctx, req.ToID, req.ToUserID, req.Amount)
	})
}
