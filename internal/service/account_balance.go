package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountBalanceService struct {
	pool *pgxpool.Pool
}

func NewAccountBalanceService(pool *pgxpool.Pool) *AccountBalanceService {
	return &AccountBalanceService{pool: pool}
}

func (s *AccountBalanceService) GetAccountBalance(ctx context.Context, id string, userID string) (model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.pool)
	return repo.FindByID(ctx, id, userID)
}

func (s *AccountBalanceService) GetAccountBalances(ctx context.Context, accountID string, userID string) ([]model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.pool)
	return repo.FindByAccountID(ctx, accountID, userID)
}

func (s *AccountBalanceService) CreateOrUpdateAccountBalance(ctx context.Context, ab model.AccountBalance) (model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.pool)
	return repo.Upsert(ctx, ab)
}

func (s *AccountBalanceService) DeleteAccountBalance(ctx context.Context, id string, userID string) (model.AccountBalance, error) {
	repo := repository.NewAccountBalanceRepository(s.pool)
	return repo.Delete(ctx, id, userID)
}
