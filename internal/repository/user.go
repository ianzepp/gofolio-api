package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type UserRepository struct {
	db db.DBTX
}

func NewUserRepository(db db.DBTX) *UserRepository {
	return &UserRepository{db: db}
}

const userColumns = `id, "accessToken", "authChallenge", "createdAt",
	provider, role, "thirdPartyId", "updatedAt"`

func (r *UserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+`
		 FROM "User"
		 WHERE id = $1`, id)
	if err != nil {
		return model.User{}, fmt.Errorf("query user: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.User])
}

func (r *UserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "User" (id, "accessToken", provider, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+userColumns,
		user.ID, user.AccessToken, user.Provider, user.Role)
	if err != nil {
		return model.User{}, fmt.Errorf("insert user: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.User])
}

func (r *UserRepository) FindByAccessToken(ctx context.Context, accessToken string) (model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+`
		 FROM "User"
		 WHERE "accessToken" = $1`, accessToken)
	if err != nil {
		return model.User{}, fmt.Errorf("query user by access token: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.User])
}

func (r *UserRepository) FindAll(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+userColumns+`
		 FROM "User"
		 ORDER BY "createdAt" DESC`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.User])
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "User" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateAccessToken(ctx context.Context, id string, hashedAccessToken string) error {
	rows, err := r.db.Query(ctx,
		`UPDATE "User"
		 SET "accessToken" = $2
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, hashedAccessToken)
	if err != nil {
		return fmt.Errorf("update user access token: %w", err)
	}
	_, err = pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.User])
	if err != nil {
		return fmt.Errorf("update user access token: %w", err)
	}
	return nil
}

func (r *UserRepository) FindSettings(ctx context.Context, userID string) (model.Settings, error) {
	rows, err := r.db.Query(ctx,
		`SELECT "userId", settings, "updatedAt"
		 FROM "Settings"
		 WHERE "userId" = $1`, userID)
	if err != nil {
		return model.Settings{}, fmt.Errorf("query user settings: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Settings])
}

func (r *UserRepository) UpsertSettings(ctx context.Context, s model.Settings) (model.Settings, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "Settings" ("userId", settings)
		 VALUES ($1, $2)
		 ON CONFLICT ("userId") DO UPDATE SET settings = $2
		 RETURNING "userId", settings, "updatedAt"`,
		s.UserId, s.Settings)
	if err != nil {
		return model.Settings{}, fmt.Errorf("upsert user settings: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Settings])
}
