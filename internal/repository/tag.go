package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type TagRepository struct {
	db db.DBTX
}

func NewTagRepository(db db.DBTX) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) FindAll(ctx context.Context) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, "userId"
		 FROM "Tag"
		 ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Tag])
}

func (r *TagRepository) FindAllWithActivityCount(ctx context.Context) ([]model.TagWithActivityCount, error) {
	rows, err := r.db.Query(ctx,
		`SELECT t.id,
		        t.name,
		        t."userId",
		        COUNT(ot."A")::int AS "activityCount"
		 FROM "Tag" t
		 LEFT JOIN "_OrderToTag" ot ON ot."B" = t.id
		 GROUP BY t.id, t.name, t."userId"
		 ORDER BY t.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query tags with activity count: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.TagWithActivityCount])
}

func (r *TagRepository) FindByID(ctx context.Context, id string) (model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, "userId"
		 FROM "Tag"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Tag{}, fmt.Errorf("query tag: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Tag])
}

func (r *TagRepository) FindByUserID(ctx context.Context, userID string) ([]model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, "userId"
		 FROM "Tag"
		 WHERE "userId" = $1
		 ORDER BY name ASC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query tags by user: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Tag])
}

func (r *TagRepository) Create(ctx context.Context, tag model.Tag) (model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "Tag" (id, name, "userId")
		 VALUES ($1, $2, $3)
		 RETURNING id, name, "userId"`,
		tag.ID, tag.Name, tag.UserId)
	if err != nil {
		return model.Tag{}, fmt.Errorf("insert tag: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Tag])
}

func (r *TagRepository) Update(ctx context.Context, id string, name string) (model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE "Tag" SET name = $2
		 WHERE id = $1
		 RETURNING id, name, "userId"`,
		id, name)
	if err != nil {
		return model.Tag{}, fmt.Errorf("update tag: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Tag])
}

func (r *TagRepository) Delete(ctx context.Context, id string) (model.Tag, error) {
	rows, err := r.db.Query(ctx,
		`DELETE FROM "Tag"
		 WHERE id = $1
		 RETURNING id, name, "userId"`, id)
	if err != nil {
		return model.Tag{}, fmt.Errorf("delete tag: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Tag])
}
