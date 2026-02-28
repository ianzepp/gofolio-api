// middleware.go wires together credential extraction, identity resolution, and
// permission enforcement into composable http.HandlerFunc wrappers.
//
// Design: authentication and authorization are kept as separate middleware
// layers rather than combined into a single guard. This allows routes that
// only need to know the caller's identity (e.g., to scope a query) to use
// Authenticate alone, while routes with hard access requirements use
// RequirePermission. The separation also makes testing each concern in
// isolation straightforward.
//
// Error responses use a JSON envelope ({"statusCode":…,"message":…}) that
// matches the NestJS HttpException format, so API clients do not need to
// handle different error shapes between the two server implementations.
package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// Config holds the dependencies required by auth middleware. It is passed by
// value so callers can create lightweight variants (e.g., without a Pool) for
// testing or for routes that only need JWT validation.
type Config struct {
	JWTSecret string
	DB        *sqlx.DB // may be nil if DB is unavailable
}

// Authenticate validates JWT or API key credentials and stores the resolved
// user ID in the request context as an AuthInfo. Returns HTTP 401 on any
// failure; does not call next in that case.
//
// PHASE 1: JWT Bearer — stateless, no DB round-trip. Preferred path for
// interactive sessions because it is faster and works even when the DB is
// temporarily unavailable.
//
// PHASE 2: API key — hashes the supplied key and looks it up in the ApiKey
// table. Requires a live DB connection; returns 503 if cfg.DB is nil.
//
// WHY JWT first: JWT validation is CPU-only (HMAC verify) while API key
// lookup requires a DB query plus ~10ms of PBKDF2 hashing. Checking JWT
// first avoids the DB cost for the common case of browser/session clients.
func Authenticate(cfg Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// PHASE 1: JWT Bearer token — stateless, no DB hit.
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

		// PHASE 2: API key — hash then DB lookup.
		if apiKey, ok := ExtractApiKey(authHeader); ok {
			if cfg.DB == nil {
				writeError(w, http.StatusServiceUnavailable, "Service Unavailable")
				return
			}
			hashedKey, err := HashApiKey(apiKey)
			if err != nil {
				log.Printf("auth: API key hashing failed: %v", err)
				writeError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
			repo := repository.NewApiKeyRepository(cfg.DB)
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

		// No recognized credential scheme present.
		writeError(w, http.StatusUnauthorized, "Unauthorized")
	}
}

// RequirePermission is a single-permission convenience wrapper around
// RequireAnyPermission. Prefer this for routes guarded by exactly one
// permission; use RequireAnyPermission directly for OR-style checks.
func RequirePermission(cfg Config, perm string, next http.HandlerFunc) http.HandlerFunc {
	return RequireAnyPermission(cfg, []string{perm}, next)
}

// RequireAnyPermission authenticates the request, loads the full user record
// from DB, and calls next only if the user's role has at least one of the
// listed permissions (OR semantics). Returns 401 if unauthenticated, 403 if
// none of the permissions match.
//
// WHY: Some Ghostfolio endpoints accept multiple roles (e.g., both ADMIN and
// USER can read their own data). Encoding OR logic here avoids duplicating
// permission checks in individual handlers.
//
// NOTE: The user record is loaded unconditionally to obtain the role. This is
// a DB read on every protected request. If this becomes a bottleneck, the
// role could be embedded in the JWT claim and trusted without a DB lookup.
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

// authenticatedUser resolves the current request's AuthInfo into a fully
// loaded User record. It writes an error response and returns (nil, false)
// on any failure so callers can return immediately without additional checks.
//
// The loaded user is also cached back onto info.User so downstream handlers
// that call GetAuthInfo can access the full model without a second DB query.
func authenticatedUser(w http.ResponseWriter, r *http.Request, cfg Config) (*model.User, bool) {
	if cfg.DB == nil {
		writeError(w, http.StatusServiceUnavailable, "Service Unavailable")
		return nil, false
	}

	info := GetAuthInfo(r.Context())
	if info == nil {
		// EDGE: Should not happen when called from within Authenticate's closure,
		// but guarded defensively in case authenticatedUser is ever called directly.
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}

	userRepo := repository.NewUserRepository(cfg.DB)
	user, err := userRepo.FindByID(r.Context(), info.UserID)
	if err != nil {
		log.Printf("auth: failed to load user %s: %v", info.UserID, err)
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return nil, false
	}
	info.User = &user
	return &user, true
}

// writeError sends a JSON error response matching the NestJS HttpException
// envelope format so API clients see a consistent shape regardless of which
// server implementation handled the request.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"statusCode": status,
		"message":    message,
	})
}
