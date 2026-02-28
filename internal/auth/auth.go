package auth

import (
	"context"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
)

type contextKey int

const authInfoKey contextKey = iota

// AuthInfo holds the authenticated user's identity extracted from a request.
type AuthInfo struct {
	UserID string
	User   *model.User // nil until loaded by RequirePermission
}

// WithAuthInfo stores auth information in the context.
func WithAuthInfo(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey, info)
}

// GetAuthInfo retrieves auth information from the context.
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
