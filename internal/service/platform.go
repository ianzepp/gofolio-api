package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlatformService struct {
	pool *pgxpool.Pool
}

func NewPlatformService(pool *pgxpool.Pool) *PlatformService {
	return &PlatformService{pool: pool}
}

func (s *PlatformService) GetPlatforms(ctx context.Context) ([]model.Platform, error) {
	repo := repository.NewPlatformRepository(s.pool)
	return repo.FindAll(ctx)
}

func (s *PlatformService) GetPlatform(ctx context.Context, id string) (model.Platform, error) {
	repo := repository.NewPlatformRepository(s.pool)
	return repo.FindByID(ctx, id)
}

func (s *PlatformService) CreatePlatform(ctx context.Context, p model.Platform) (model.Platform, error) {
	repo := repository.NewPlatformRepository(s.pool)
	return repo.Create(ctx, p)
}

func (s *PlatformService) UpdatePlatform(ctx context.Context, p model.Platform) (model.Platform, error) {
	repo := repository.NewPlatformRepository(s.pool)
	return repo.Update(ctx, p)
}

func (s *PlatformService) DeletePlatform(ctx context.Context, id string) error {
	repo := repository.NewPlatformRepository(s.pool)
	return repo.Delete(ctx, id)
}
