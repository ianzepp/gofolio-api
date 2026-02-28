package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserService struct {
	pool *pgxpool.Pool
}

func NewUserService(pool *pgxpool.Pool) *UserService {
	return &UserService{pool: pool}
}

func (s *UserService) GetUser(ctx context.Context, id string) (model.User, error) {
	repo := repository.NewUserRepository(s.pool)
	return repo.FindByID(ctx, id)
}

func (s *UserService) GetUsers(ctx context.Context) ([]model.User, error) {
	repo := repository.NewUserRepository(s.pool)
	return repo.FindAll(ctx)
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	repo := repository.NewUserRepository(s.pool)
	return repo.Delete(ctx, id)
}

func (s *UserService) GetSettings(ctx context.Context, userID string) (model.Settings, error) {
	repo := repository.NewUserRepository(s.pool)
	return repo.FindSettings(ctx, userID)
}

func (s *UserService) UpdateSettings(ctx context.Context, s2 model.Settings) (model.Settings, error) {
	repo := repository.NewUserRepository(s.pool)
	return repo.UpsertSettings(ctx, s2)
}
