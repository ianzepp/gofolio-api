package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TagService struct {
	pool *pgxpool.Pool
}

func NewTagService(pool *pgxpool.Pool) *TagService {
	return &TagService{pool: pool}
}

func (s *TagService) GetTags(ctx context.Context) ([]model.Tag, error) {
	repo := repository.NewTagRepository(s.pool)
	return repo.FindAll(ctx)
}

func (s *TagService) GetTagsByUserID(ctx context.Context, userID string) ([]model.Tag, error) {
	repo := repository.NewTagRepository(s.pool)
	return repo.FindByUserID(ctx, userID)
}

func (s *TagService) CreateTag(ctx context.Context, tag model.Tag) (model.Tag, error) {
	repo := repository.NewTagRepository(s.pool)
	return repo.Create(ctx, tag)
}

func (s *TagService) UpdateTag(ctx context.Context, id string, name string) (model.Tag, error) {
	repo := repository.NewTagRepository(s.pool)
	return repo.Update(ctx, id, name)
}

func (s *TagService) DeleteTag(ctx context.Context, id string) error {
	repo := repository.NewTagRepository(s.pool)
	return repo.Delete(ctx, id)
}
