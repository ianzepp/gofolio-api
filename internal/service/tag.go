package service

import (
	"context"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// TagService manages user-defined tags that can be attached to activities
// (orders) to support custom categorisation and filtering in the portfolio view.
// Tags are global within a user's account but can be queried system-wide by
// admins (GetTags / GetTagsWithActivityCount) for moderation purposes.
type TagService struct {
	db *sqlx.DB
}

// NewTagService constructs a TagService backed by the given pool.
func NewTagService(db *sqlx.DB) *TagService {
	return &TagService{db: db}
}

// GetTags returns all tags in the system without activity counts, suitable for
// simple list/autocomplete endpoints where count information is not needed.
func (s *TagService) GetTags(ctx context.Context) ([]model.Tag, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.FindAll(ctx)
}

// GetTagsWithActivityCount returns all tags enriched with the number of
// activities each tag is applied to. This is used in admin views to identify
// unused tags that can be pruned.
func (s *TagService) GetTagsWithActivityCount(ctx context.Context) ([]model.TagWithActivityCount, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.FindAllWithActivityCount(ctx)
}

func (s *TagService) GetTag(ctx context.Context, id string) (model.Tag, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.FindByID(ctx, id)
}

func (s *TagService) GetTagsByUserID(ctx context.Context, userID string) ([]model.Tag, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.FindByUserID(ctx, userID)
}

func (s *TagService) CreateTag(ctx context.Context, tag model.Tag) (model.Tag, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.Create(ctx, tag)
}

func (s *TagService) UpdateTag(ctx context.Context, id string, name string) (model.Tag, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.Update(ctx, id, name)
}

func (s *TagService) DeleteTag(ctx context.Context, id string) (model.Tag, error) {
	repo := repository.NewTagRepository(s.db)
	return repo.Delete(ctx, id)
}
