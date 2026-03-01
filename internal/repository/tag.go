// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ianzepp/gofolio-api/internal/db"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/jmoiron/sqlx"
)

// TagRepository handles persistence for user-defined tags that can be applied
// to orders (via the many-to-many _OrderToTag join table). Tags let users
// group and filter portfolio activity by arbitrary labels (e.g. "dividend",
// "retirement", "crypto"). Tags may be global (userId IS NULL) or user-scoped.
type TagRepository struct {
	db db.DBTX
}

// NewTagRepository constructs a TagRepository backed by the provided handle.
func NewTagRepository(db db.DBTX) *TagRepository {
	return &TagRepository{db: db}
}

// FindAll returns every tag in the system (global and user-scoped), sorted
// alphabetically. Used by admin endpoints and export routines.
func (r *TagRepository) FindAll(ctx context.Context) ([]model.Tag, error) {
	var items []model.Tag
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT id, name, "userId"
		 FROM "Tag"
		 ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	return items, nil
}

// FindAllWithActivityCount returns every tag enriched with a count of the
// orders that reference it. The count is derived from the _OrderToTag pivot
// table and is cast to int to avoid pgx uint64 mapping issues.
// WHY: the tag management UI shows usage counts to help users identify
// orphaned tags before deleting them.
func (r *TagRepository) FindAllWithActivityCount(ctx context.Context) ([]model.TagWithActivityCount, error) {
	var items []model.TagWithActivityCount
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT t.id,
		        t.name,
		        t."userId",
		        CAST(COUNT(ot."A") AS INTEGER) AS "activityCount"
		 FROM "Tag" t
		 LEFT JOIN "_OrderToTag" ot ON ot."B" = t.id
		 GROUP BY t.id, t.name, t."userId"
		 ORDER BY t.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query tags with activity count: %w", err)
	}
	return items, nil
}

// FindByID fetches a single tag by its primary key. Returns ErrNoRows when no
// matching record exists.
func (r *TagRepository) FindByID(ctx context.Context, id string) (model.Tag, error) {
	var item model.Tag
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT id, name, "userId"
		 FROM "Tag"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Tag{}, fmt.Errorf("query tag: %w", err)
	}
	return item, nil
}

// FindByUserID returns tags owned by userID, sorted alphabetically.
// Global tags (userId IS NULL) are not included — callers that need both
// should call FindAll and filter client-side.
func (r *TagRepository) FindByUserID(ctx context.Context, userID string) ([]model.Tag, error) {
	var items []model.Tag
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT id, name, "userId"
		 FROM "Tag"
		 WHERE "userId" = $1
		 ORDER BY name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query tags by user: %w", err)
	}
	return items, nil
}

// Create inserts a new tag and returns the persisted record. The caller must
// supply a pre-generated ID.
func (r *TagRepository) Create(ctx context.Context, tag model.Tag) (model.Tag, error) {
	var item model.Tag
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`INSERT INTO "Tag" (id, name, "userId")
		 VALUES ($1, $2, $3)
		 RETURNING id, name, "userId"`,
		tag.ID, tag.Name, tag.UserId)
	if err != nil {
		return model.Tag{}, fmt.Errorf("insert tag: %w", err)
	}
	return item, nil
}

// Update renames a tag identified by id. Name is the only mutable field on a
// Tag — userId and id are immutable after creation.
func (r *TagRepository) Update(ctx context.Context, id string, name string) (model.Tag, error) {
	var item model.Tag
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`UPDATE "Tag" SET name = $2
		 WHERE id = $1
		 RETURNING id, name, "userId"`,
		id, name)
	if err != nil {
		return model.Tag{}, fmt.Errorf("update tag: %w", err)
	}
	return item, nil
}

// Delete removes a tag by ID and returns the deleted record so the caller can
// confirm which tag was affected without a separate lookup. The database
// cascades the deletion to the _OrderToTag pivot rows automatically.
func (r *TagRepository) Delete(ctx context.Context, id string) (model.Tag, error) {
	var item model.Tag
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`DELETE FROM "Tag"
		 WHERE id = $1
		 RETURNING id, name, "userId"`, id)
	if err != nil {
		return model.Tag{}, fmt.Errorf("delete tag: %w", err)
	}
	return item, nil
}
