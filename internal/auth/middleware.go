package auth

import (
	"log"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/handler"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds auth middleware dependencies.
type Config struct {
	JWTSecret string
	Pool      *pgxpool.Pool // may be nil if DB is unavailable
}

// Authenticate validates JWT or API key credentials and stores the user ID
// in the request context. Returns 401 on failure.
// JWT path is stateless (no DB hit). API key path hashes the key and looks
// it up in the ApiKey table.
func Authenticate(cfg Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Try JWT Bearer token first.
		if token, ok := ExtractBearerToken(authHeader); ok {
			userID, err := ParseJWT(token, cfg.JWTSecret)
			if err != nil {
				log.Printf("auth: invalid JWT: %v", err)
				handler.Error(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			ctx := WithAuthInfo(r.Context(), &AuthInfo{UserID: userID})
			next(w, r.WithContext(ctx))
			return
		}

		// Try API key.
		if apiKey, ok := ExtractApiKey(authHeader); ok {
			if cfg.Pool == nil {
				handler.Error(w, http.StatusServiceUnavailable, "Service Unavailable")
				return
			}
			hashedKey := HashApiKey(apiKey)
			repo := repository.NewApiKeyRepository(cfg.Pool)
			userID, err := repo.FindUserIDByHashedKey(r.Context(), hashedKey)
			if err != nil {
				log.Printf("auth: invalid API key: %v", err)
				handler.Error(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			ctx := WithAuthInfo(r.Context(), &AuthInfo{UserID: userID})
			next(w, r.WithContext(ctx))
			return
		}

		handler.Error(w, http.StatusUnauthorized, "Unauthorized")
	}
}

// RequirePermission authenticates the request, loads the user from DB to
// determine their role, and checks that the role has the required permission.
// Returns 401 if unauthenticated, 403 if the permission check fails.
func RequirePermission(cfg Config, perm string, next http.HandlerFunc) http.HandlerFunc {
	return Authenticate(cfg, func(w http.ResponseWriter, r *http.Request) {
		if cfg.Pool == nil {
			handler.Error(w, http.StatusServiceUnavailable, "Service Unavailable")
			return
		}

		info := GetAuthInfo(r.Context())
		if info == nil {
			handler.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		userRepo := repository.NewUserRepository(cfg.Pool)
		user, err := userRepo.FindByID(r.Context(), info.UserID)
		if err != nil {
			log.Printf("auth: failed to load user %s: %v", info.UserID, err)
			handler.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		info.User = &user

		if !HasPermission(user.Role, perm) {
			handler.Error(w, http.StatusForbidden, "Forbidden")
			return
		}

		next(w, r)
	})
}
