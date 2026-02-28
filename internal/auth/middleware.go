package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
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
				writeError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			ctx := WithAuthInfo(r.Context(), &AuthInfo{UserID: userID})
			next(w, r.WithContext(ctx))
			return
		}

		// Try API key.
		if apiKey, ok := ExtractApiKey(authHeader); ok {
			if cfg.Pool == nil {
				writeError(w, http.StatusServiceUnavailable, "Service Unavailable")
				return
			}
			hashedKey := HashApiKey(apiKey)
			repo := repository.NewApiKeyRepository(cfg.Pool)
			userID, err := repo.FindUserIDByHashedKey(r.Context(), hashedKey)
			if err != nil {
				log.Printf("auth: invalid API key: %v", err)
				writeError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			ctx := WithAuthInfo(r.Context(), &AuthInfo{UserID: userID})
			next(w, r.WithContext(ctx))
			return
		}

		writeError(w, http.StatusUnauthorized, "Unauthorized")
	}
}

// RequirePermission authenticates the request, loads the user from DB to
// determine their role, and checks that the role has the required permission.
// Returns 401 if unauthenticated, 403 if the permission check fails.
func RequirePermission(cfg Config, perm string, next http.HandlerFunc) http.HandlerFunc {
	return RequireAnyPermission(cfg, []string{perm}, next)
}

// RequireAnyPermission authenticates the request, loads the user from DB,
// and checks whether the user's role has at least one required permission.
// Returns 401 if unauthenticated, 403 if the permission check fails.
func RequireAnyPermission(cfg Config, perms []string, next http.HandlerFunc) http.HandlerFunc {
	return Authenticate(cfg, func(w http.ResponseWriter, r *http.Request) {
		user, ok := authenticatedUser(w, r, cfg)
		if !ok {
			return
		}
		for _, perm := range perms {
			if HasPermission(user.Role, perm) {
				next(w, r)
				return
			}
		}
		writeError(w, http.StatusForbidden, "Forbidden")
	})
}

func authenticatedUser(w http.ResponseWriter, r *http.Request, cfg Config) (*model.User, bool) {
	if cfg.Pool == nil {
		writeError(w, http.StatusServiceUnavailable, "Service Unavailable")
		return nil, false
	}

	info := GetAuthInfo(r.Context())
	if info == nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}

	userRepo := repository.NewUserRepository(cfg.Pool)
	user, err := userRepo.FindByID(r.Context(), info.UserID)
	if err != nil {
		log.Printf("auth: failed to load user %s: %v", info.UserID, err)
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}
	info.User = &user
	return &user, true
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"statusCode": status,
		"message":    message,
	})
}
