package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/jackc/pgx/v5"
)

type ApiKeyRepository struct {
	db db.DBTX
}

func NewApiKeyRepository(db db.DBTX) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

// FindUserIDByHashedKey looks up the user ID associated with a hashed API key.
func (r *ApiKeyRepository) FindUserIDByHashedKey(ctx context.Context, hashedKey string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx,
		`SELECT "userId" FROM "ApiKey" WHERE "hashedKey" = $1`,
		hashedKey).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("api key not found")
		}
		return "", fmt.Errorf("query api key: %w", err)
	}
	return userID, nil
}
