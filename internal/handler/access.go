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
	"encoding/json"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
)

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
			Alias         *string          `json:"alias"`
			GranteeUserId *string          `json:"granteeUserId"`
			Permissions   []string         `json:"permissions"`
			Settings      json.RawMessage  `json:"settings"`
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

func DeleteAccess(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		if err := deps.AccessService.DeleteAccess(r.Context(), id); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

func UpdateAccess(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		var body struct {
			Alias       *string          `json:"alias"`
			Permissions []string         `json:"permissions"`
			Settings    json.RawMessage  `json:"settings"`
		}
		if !Bind(w, r, &body) {
			return
		}

		updated, err := deps.AccessService.UpdateAccess(r.Context(), id, body.Alias, body.Permissions, body.Settings)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}
