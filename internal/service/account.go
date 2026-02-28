package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountService struct {
	pool *pgxpool.Pool
}

func NewAccountService(pool *pgxpool.Pool) *AccountService {
	return &AccountService{pool: pool}
}

func (s *AccountService) GetAccounts(ctx context.Context, userID string) ([]model.Account, error) {
	repo := repository.NewAccountRepository(s.pool)
	return repo.FindByUserID(ctx, userID)
}

func (s *AccountService) GetAccount(ctx context.Context, id string, userID string) (model.Account, error) {
	repo := repository.NewAccountRepository(s.pool)
	return repo.FindByID(ctx, id, userID)
}

func (s *AccountService) CreateAccount(ctx context.Context, acct model.Account) (model.Account, error) {
	repo := repository.NewAccountRepository(s.pool)
	return repo.Create(ctx, acct)
}

func (s *AccountService) UpdateAccount(ctx context.Context, acct model.Account) (model.Account, error) {
	repo := repository.NewAccountRepository(s.pool)
	return repo.Update(ctx, acct)
}

func (s *AccountService) DeleteAccount(ctx context.Context, id string, userID string) error {
	repo := repository.NewAccountRepository(s.pool)
	return repo.Delete(ctx, id, userID)
}

type TransferRequest struct {
	FromID     string
	FromUserID string
	ToID       string
	ToUserID   string
	Amount     float64
}

func (s *AccountService) TransferBalance(ctx context.Context, req TransferRequest) error {
	return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := repository.NewAccountRepository(tx)
		if err := repo.UpdateBalance(ctx, req.FromID, req.FromUserID, -req.Amount); err != nil {
			return err
		}
		return repo.UpdateBalance(ctx, req.ToID, req.ToUserID, req.Amount)
	})
}
