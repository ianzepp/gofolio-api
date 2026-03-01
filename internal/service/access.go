package service

import (
	"context"
	"encoding/json"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// AccessService manages the Access grants that allow one user to view another
// user's portfolio with restricted permissions. An Access record carries an
// alias (display name for the grantee), a permissions list, and optional
// per-access settings stored as JSONB. Ownership is enforced via userID
// scoping on all mutating operations.
type AccessService struct {
	db *sqlx.DB
}

// NewAccessService constructs an AccessService backed by the given pool.
func NewAccessService(db *sqlx.DB) *AccessService {
	return &AccessService{db: db}
}

// GetAccesses returns all access grants where the given user is the grantor.
func (s *AccessService) GetAccesses(ctx context.Context, userID string) ([]model.Access, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.FindByUserID(ctx, userID)
}

func (s *AccessService) GetAccess(ctx context.Context, id string) (model.Access, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.FindByID(ctx, id)
}

func (s *AccessService) GetAccessByUserID(ctx context.Context, id string, userID string) (model.Access, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.FindByIDAndUserID(ctx, id, userID)
}

func (s *AccessService) CreateAccess(ctx context.Context, a model.Access) (model.Access, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.Create(ctx, a)
}

// UpdateAccess updates an access grant by ID without user-scoping, intended
// for admin-level operations. For user-scoped updates see UpdateAccessByUserID.
func (s *AccessService) UpdateAccess(ctx context.Context, id string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.Update(ctx, id, alias, permissions, settings)
}

func (s *AccessService) UpdateAccessByUserID(ctx context.Context, id string, userID string, alias *string, permissions []string, settings json.RawMessage) (model.Access, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.UpdateByUserID(ctx, id, userID, alias, permissions, settings)
}

func (s *AccessService) DeleteAccess(ctx context.Context, id string) error {
	repo := repository.NewAccessRepository(s.db)
	return repo.Delete(ctx, id)
}

// DeleteAccessByUserID removes an access grant only if it belongs to the given
// user, returning the number of rows deleted so callers can distinguish
// "not found" (0) from "deleted" (1) without a prior existence check.
func (s *AccessService) DeleteAccessByUserID(ctx context.Context, id string, userID string) (int64, error) {
	repo := repository.NewAccessRepository(s.db)
	return repo.DeleteByUserID(ctx, id, userID)
}
