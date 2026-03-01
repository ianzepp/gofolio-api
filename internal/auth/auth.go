// Package auth implements request authentication and authorization for the
// Ghostfolio Go API server. It supports two credential schemes:
//
//   - JWT Bearer tokens: stateless, validated using HMAC-SHA256 against the
//     server's JWTSecret. No database round-trip required.
//   - API keys: stateful, hashed with PBKDF2-SHA256 and looked up in the
//     ApiKey table. Requires a live DB connection.
//
// Authentication is layered: the Authenticate middleware identifies the caller
// and stores an AuthInfo in the request context. Authorization (RequirePermission
// / RequireAnyPermission) then loads the full user record and enforces role-based
// access control defined in permissions.go.
//
// Context propagation uses a package-private contextKey to avoid collisions with
// other packages storing values in context.Context.
package auth

import (
	"context"
	"net/http"

	"github.com/ianzepp/gofolio-api/internal/model"
)

// WHY: A typed int key prevents accidental collisions when multiple packages
// store values in a context.Context using plain string or int keys.
type contextKey int

const authInfoKey contextKey = iota

// AuthInfo holds the authenticated caller's identity extracted from a request.
//
// UserID is always populated after successful authentication. User is lazily
// loaded by RequirePermission / RequireAnyPermission — it is nil for routes
// that only call Authenticate without a permission check. Callers must not
// assume User is non-nil without going through a RequirePermission wrapper.
type AuthInfo struct {
	UserID string
	User   *model.User // nil until loaded by RequirePermission
}

// WithAuthInfo stores auth information in the request context. It is called
// by the Authenticate middleware immediately after credential validation, so
// that downstream handlers and middleware can retrieve the identity without
// re-parsing or re-validating the credential.
func WithAuthInfo(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey, info)
}

// GetAuthInfo retrieves auth information from the context. Returns nil when
// the request was not routed through the Authenticate middleware, which can
// happen on public endpoints. Callers that require authentication should use
// RequirePermission rather than checking this value manually.
func GetAuthInfo(ctx context.Context) *AuthInfo {
	info, _ := ctx.Value(authInfoKey).(*AuthInfo)
	return info
}

// UserID is a convenience function that returns the authenticated user's ID
// from the request context, or "" if unauthenticated.
func UserID(r *http.Request) string {
	if info := GetAuthInfo(r.Context()); info != nil {
		return info.UserID
	}
	return ""
}
