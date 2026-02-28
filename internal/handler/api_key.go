package handler

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/google/uuid"
)

// CreateApiKey handles POST /api/v1/api-keys.
// Generates a random API key, hashes it with PBKDF2-SHA256, replaces any
// existing key for the user, and returns the plaintext key (shown only once).
// Auth: jwt + createApiKey permission, enforced at the routing layer.
func CreateApiKey(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.DB == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		// Generate random API key: 32 random hex chars in groups of 4
		rawKey, err := generateApiKey()
		if err != nil {
			Error(w, http.StatusInternalServerError, "failed to generate API key")
			return
		}

		hashedKey, err := auth.HashApiKey(rawKey)
		if err != nil {
			Error(w, http.StatusInternalServerError, "failed to hash API key")
			return
		}

		repo := repository.NewApiKeyRepository(deps.DB)

		// One key per user — delete any existing key first
		if err := repo.DeleteByUserID(r.Context(), userID); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		id := uuid.New().String()
		if err := repo.Create(r.Context(), id, hashedKey, userID); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusCreated, map[string]any{
			"apiKey": rawKey,
		})
	}
}

// generateApiKey creates a random API key formatted as 8 groups of 4 hex
// chars separated by dashes (e.g., "a1b2-c3d4-e5f6-..."), matching the
// NestJS ApiKeyService format.
func generateApiKey() (string, error) {
	b := make([]byte, 16) // 16 bytes = 32 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	hex := fmt.Sprintf("%x", b)
	// Format as xxxx-xxxx-xxxx-xxxx-xxxx-xxxx-xxxx-xxxx
	parts := make([]string, 8)
	for i := 0; i < 8; i++ {
		parts[i] = hex[i*4 : (i+1)*4]
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "-" + p
	}
	return result, nil
}
