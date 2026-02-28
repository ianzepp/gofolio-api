package service

import (
	"context"
	"encoding/json"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessService struct {
	pool *pgxpool.Pool
}

func NewAccessService(pool *pgxpool.Pool) *AccessService {
	return &AccessService{pool: pool}
}

func (s *AccessService) GetAccesses(ctx context.Context, userID string) ([]model.Access, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.FindByUserID(ctx, userID)
}

func (s *AccessService) GetAccess(ctx context.Context, id string) (model.Access, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.FindByID(ctx, id)
}

func (s *AccessService) GetAccessByUserID(ctx context.Context, id string, userID string) (model.Access, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.FindByIDAndUserID(ctx, id, userID)
}

func (s *AccessService) CreateAccess(ctx context.Context, a model.Access) (model.Access, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.Create(ctx, a)
}

func (s *AccessService) UpdateAccess(ctx context.Context, id string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.Update(ctx, id, alias, permissions, settings)
}

func (s *AccessService) UpdateAccessByUserID(ctx context.Context, id string, userID string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.UpdateByUserID(ctx, id, userID, alias, permissions, settings)
}

func (s *AccessService) DeleteAccess(ctx context.Context, id string) error {
	repo := repository.NewAccessRepository(s.pool)
	return repo.Delete(ctx, id)
}

func (s *AccessService) DeleteAccessByUserID(ctx context.Context, id string, userID string) (int64, error) {
	repo := repository.NewAccessRepository(s.pool)
	return repo.DeleteByUserID(ctx, id, userID)
}
