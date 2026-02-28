// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/jackc/pgx/v5"
)

// ApiKeyRepository handles persistence for the ApiKey table, which stores
// hashed API keys that grant programmatic access to the Ghostfolio API on
// behalf of a specific user. Only the SHA-256 hash of each key is stored —
// the plaintext is shown to the user once at creation and then discarded.
//
// This repository intentionally has a minimal surface: API key reads happen
// on every authenticated request, so only the fast userId-lookup path is
// exposed here. Key creation and deletion are handled by the service layer
// directly.
type ApiKeyRepository struct {
	db db.DBTX
}

// NewApiKeyRepository constructs an ApiKeyRepository backed by the provided
// database handle.
func NewApiKeyRepository(db db.DBTX) *ApiKeyRepository {
	return &ApiKeyRepository{db: db}
}

// FindUserIDByHashedKey resolves the userId associated with a hashed API key.
// The caller must hash the raw inbound key before calling here — storing and
// comparing only hashes prevents key exposure in logs, database backups, and
// memory dumps.
// Returns a sentinel error string ("api key not found") for ErrNoRows rather
// than wrapping it, intentionally avoiding leaking whether the key ever
// existed.
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
