// Package handler — access.go implements handlers for /api/v1/access.
//
// Access records represent portfolio-sharing grants: a user (grantor) can share
// their portfolio with another user (grantee) with a specific permission set.
// This enables the Ghostfolio "public portfolio" and "shared view" features.
//
// Ownership is enforced by scoping all mutations to the authenticated user's ID.
// Delete and update operations return 403 (not 404) when the target record does
// not exist or is not owned by the caller, to avoid leaking ID existence.
//
// Parity reference: apps/api/src/app/access/access.controller.ts
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/access/access.controller.ts

Routes + request/response shape:
- GET /api/v1/access (GetAccesses)
  Auth/permission: jwt / -
  Response: Access[] (unwrapped array)
- POST /api/v1/access (CreateAccess)
  Auth/permission: jwt / permissions.createAccess
  Response: Access
- DELETE /api/v1/access/{id} (DeleteAccess)
  Auth/permission: jwt / permissions.deleteAccess
  Response: Access
- PUT /api/v1/access/{id} (UpdateAccess)
  Auth/permission: jwt / permissions.updateAccess
  Response: Access
*/

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
)

// GetAccesses handles GET /api/v1/access and returns the caller's outbound
// access grants as an unwrapped array (no envelope key).
func GetAccesses(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		accesses, err := deps.AccessService.GetAccesses(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if accesses == nil {
			accesses = []model.Access{}
		}
		// Legacy parity: returns unwrapped array
		JSON(w, http.StatusOK, accesses)
	}
}

// CreateAccess handles POST /api/v1/access. Permissions and settings default to
// empty slice/object if absent in the body so the DB schema invariants are met
// without requiring the client to explicitly send empty values.
func CreateAccess(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			Alias         *string         `json:"alias"`
			GranteeUserId *string         `json:"granteeUserId"`
			Permissions   []string        `json:"permissions"`
			Settings      json.RawMessage `json:"settings"`
		}
		if !Bind(w, r, &body) {
			return
		}

		if body.Permissions == nil {
			body.Permissions = []string{}
		}
		if body.Settings == nil {
			body.Settings = json.RawMessage(`{}`)
		}

		access := model.Access{
			ID:            uuid.New().String(),
			Alias:         body.Alias,
			GranteeUserId: body.GranteeUserId,
			Permissions:   body.Permissions,
			Settings:      body.Settings,
			UserId:        userID,
		}

		created, err := deps.AccessService.CreateAccess(r.Context(), access)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, created)
	}
}

// DeleteAccess handles DELETE /api/v1/access/{id}. The service scopes the delete
// to the caller's userID; a zero row-count result (not found or not owned) is
// treated as 403.
func DeleteAccess(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		count, err := deps.AccessService.DeleteAccessByUserID(r.Context(), id, userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if count == 0 {
			// Legacy parity: non-owned/non-existent access delete returns forbidden.
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// UpdateAccess handles PUT /api/v1/access/{id}. Not-found or not-owned returns
// 403 via pgx.ErrNoRows from the user-scoped update query.
func UpdateAccess(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		var body struct {
			Alias       *string         `json:"alias"`
			Permissions []string        `json:"permissions"`
			Settings    json.RawMessage `json:"settings"`
		}
		if !Bind(w, r, &body) {
			return
		}

		if body.Permissions == nil {
			body.Permissions = []string{}
		}
		if body.Settings == nil {
			body.Settings = json.RawMessage(`{}`)
		}

		updated, err := deps.AccessService.UpdateAccessByUserID(r.Context(), id, userID, body.Alias, body.Permissions, body.Settings)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Legacy parity: non-owned/non-existent access update returns forbidden.
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}
