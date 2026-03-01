// Package handler contains all HTTP handlers for the api-go server and the
// shared utilities they depend on.
//
// # File layout
//
//   - handler.go  — shared response/request helpers (this file)
//   - deps.go     — Deps struct: injected services passed into every handler
//   - health.go   — GET /api/v1/health*
//   - One file per NestJS controller domain (account.go, order.go, …)
//
// # Handler pattern
//
// Handlers are factory functions that close over *Deps and return an
// http.HandlerFunc. This avoids global state and makes handlers trivially
// testable by injecting mock services. Example:
//
//	func GetOrders(deps *Deps) http.HandlerFunc {
//	    return func(w http.ResponseWriter, r *http.Request) { ... }
//	}
//
// Stub handlers (not yet implemented) are plain functions that return an
// empty JSON object so that the routing table compiles and differential
// tests can detect when a route goes from stub to real.
//
// # NestJS parity
//
// Error shape, pagination defaults, and filter query-parameter names are
// kept identical to the NestJS implementation so that the differential test
// suite can diff responses without field-name translation.
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ianzepp/gofolio-api/internal/auth"
)

var errResolveUserUnauthorized = errors.New("unauthorized")
var errResolveUserForbidden = errors.New("forbidden")

// --- Response helpers ---

// JSON encodes v as JSON and writes it to w with the given HTTP status code.
// Content-Type is set before WriteHeader so that the header is included even
// when status is not 200. Encoding errors are logged but not returned to the
// caller — by that point WriteHeader has already been called and the response
// cannot be changed.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("handler: failed to encode JSON response: %v", err)
	}
}

// Error writes a JSON error response in NestJS HttpException format:
//
//	{ "statusCode": 403, "message": "Forbidden" }
//
// WHY: The NestJS frontend and differential tests expect this exact shape.
// Deviating from it would require client-side workarounds that complicate the
// parity migration.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]any{
		"statusCode": status,
		"message":    message,
	})
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// HandleNotFound returns a JSON 404 for any unmatched route.
func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusNotFound, map[string]any{
		"error":      "Not Found",
		"message":    "Cannot " + r.Method + " " + r.URL.RequestURI(),
		"statusCode": http.StatusNotFound,
	})
}

// --- Request helpers ---

// Bind decodes the JSON request body into target. It returns true on success.
// On failure it writes a 400 error response and returns false, so callers can
// use the pattern:
//
//	if !Bind(w, r, &body) { return }
//
// EDGE: A nil Body (e.g. GET with no body) is treated as a missing body error
// rather than a JSON parse error, giving a clearer client-facing message.
func Bind(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		Error(w, http.StatusBadRequest, "Request body is required")
		return false
	}

	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return false
	}

	return true
}

// --- Query parameter helpers ---

// Filters captures the cross-cutting query parameters accepted by most list
// and portfolio endpoints. Fields map 1-to-1 to the query parameter names used
// by the NestJS API to preserve wire compatibility with existing frontends and
// the differential test suite.
type Filters struct {
	Accounts     []string
	AssetClasses []string
	DataSource   string
	Symbol       string
	Tags         []string
	Range        string // "1d", "1y", "5y", "max", "mtd", "wtd", "ytd"
	Query        string // free text search
	HoldingType  string
}

// ParseFilters extracts the common filter query parameters from a request.
func ParseFilters(r *http.Request) Filters {
	// Legacy parity note:
	// NestJS controllers pass query fields through as comma-separated strings
	// for list filters (accounts, assetClasses, tags). Keep this CSV parsing
	// behavior stable for differential tests and existing frontend requests.
	q := r.URL.Query()
	return Filters{
		Accounts:     splitCSV(q.Get("accounts")),
		AssetClasses: splitCSV(q.Get("assetClasses")),
		DataSource:   q.Get("dataSource"),
		Symbol:       q.Get("symbol"),
		Tags:         splitCSV(q.Get("tags")),
		Range:        q.Get("range"),
		Query:        q.Get("query"),
		HoldingType:  q.Get("holdingType"),
	}
}

// Pagination represents the cursor-less skip/take pagination used by Ghostfolio
// list endpoints. SortColumn and SortDirection are passed through to the service
// layer for SQL ORDER BY construction; they are not validated here.
type Pagination struct {
	Skip          int
	Take          int
	SortColumn    string
	SortDirection string // "asc" or "desc"
}

// ParsePagination extracts pagination query parameters from a request.
func ParsePagination(r *http.Request) Pagination {
	// Legacy parity note:
	// Default take=50 mirrors the current NestJS endpoint conventions used by
	// list endpoints that accept skip/take.
	q := r.URL.Query()
	return Pagination{
		Skip:          queryInt(q.Get("skip"), 0),
		Take:          queryInt(q.Get("take"), 50),
		SortColumn:    q.Get("sortColumn"),
		SortDirection: q.Get("sortDirection"),
	}
}

// Impersonation returns the impersonation user ID from the Impersonation-Id
// header, or an empty string if the header is absent. Admin handlers use this
// to act on behalf of another user without switching the authenticated identity.
func Impersonation(r *http.Request) string {
	// Legacy parity note:
	// Ghostfolio uses HEADER_KEY_IMPERSONATION = "Impersonation-Id" in libs/common.
	// net/http header lookup is case-insensitive, so this matches legacy clients.
	return r.Header.Get("Impersonation-Id")
}

// --- Internal utilities ---

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func queryInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// resolveUserID returns the effective user ID, considering impersonation.
// If the Impersonation-Id header is set, the caller must have the
// ImpersonateAllUsers permission (matching legacy NestJS behavior where the
// ImpersonationService validates via the Access table / admin role).
func resolveUserID(r *http.Request, deps *Deps) (string, error) {
	userID := auth.UserID(r)
	if userID == "" {
		return "", errResolveUserUnauthorized
	}

	impersonationID := Impersonation(r)
	if impersonationID == "" {
		return userID, nil
	}

	if deps == nil || deps.UserService == nil {
		return "", errResolveUserForbidden
	}
	user, err := deps.UserService.GetUser(r.Context(), userID)
	if err != nil {
		return "", errResolveUserForbidden
	}
	if !auth.HasPermission(user.Role, auth.ImpersonateAllUsers) {
		return "", errResolveUserForbidden
	}

	return impersonationID, nil
}

func writeResolveUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errResolveUserUnauthorized):
		Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	case errors.Is(err, errResolveUserForbidden):
		Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
	default:
		Error(w, http.StatusInternalServerError, err.Error())
	}
}
