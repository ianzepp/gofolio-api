// Package handler — platform.go implements handlers for /api/v1/platform and
// /api/v1/platforms.
//
// Platforms represent brokerage/broker institutions (e.g. "Interactive Brokers").
// Two routes share the same underlying data but differ in response shape:
//   - GET /api/v1/platform  → array with accountCount field (admin view)
//   - GET /api/v1/platforms → { platforms: [...] } without accountCount (public view)
//
// The accountCount field in the admin response is currently hard-coded to 0
// because computing it requires an accounts JOIN that is not yet exposed by the
// platform service. This is a known parity gap and is noted inline.
//
// Non-existent platform IDs return 403 (delete) or 400 (update) rather than 404,
// mirroring the NestJS behaviour.
//
// Parity reference:
//   - apps/api/src/app/platform/platform.controller.ts
//   - apps/api/src/app/endpoints/platforms/platforms.controller.ts
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/platform/platform.controller.ts
- apps/api/src/app/endpoints/platforms/platforms.controller.ts

Routes + request/response shape:
- GET /api/v1/platform (GetPlatformsWithAccountCount)
  Auth/permission: jwt / permissions.readPlatformsWithAccountCount
  Response: Platform[] (with accountCount field)
- POST /api/v1/platform (CreatePlatform)
  Auth/permission: jwt / permissions.createPlatform
  Response: Platform
- DELETE /api/v1/platform/{id} (DeletePlatform)
  Auth/permission: jwt / permissions.deletePlatform
  Response: Platform
- PUT /api/v1/platform/{id} (UpdatePlatform)
  Auth/permission: jwt / permissions.updatePlatform
  Response: Platform
- GET /api/v1/platforms (GetPlatforms)
  Auth/permission: jwt / permissions.readPlatforms
  Response: { platforms: Platform[] }
*/

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
	"database/sql"
)

// GetPlatformsWithAccountCount handles GET /api/v1/platform. Emits accountCount=0
// for all platforms until the account-count JOIN is implemented in the service layer.
func GetPlatformsWithAccountCount(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PlatformService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		platforms, err := deps.PlatformService.GetPlatforms(r.Context())
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if platforms == nil {
			platforms = []model.Platform{}
		}
		// Legacy parity: /platform returns array with accountCount.
		// accountCount requires a join; return 0 until wired.
		result := make([]map[string]any, len(platforms))
		for i, p := range platforms {
			result[i] = map[string]any{
				"id":           p.ID,
				"name":         p.Name,
				"url":          p.Url,
				"accountCount": 0,
			}
		}
		JSON(w, http.StatusOK, result)
	}
}

// CreatePlatform handles POST /api/v1/platform.
func CreatePlatform(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PlatformService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		var body struct {
			Name *string `json:"name"`
			Url  string  `json:"url"`
		}
		if !Bind(w, r, &body) {
			return
		}

		p := model.Platform{
			ID:   uuid.New().String(),
			Name: body.Name,
			Url:  body.Url,
		}

		created, err := deps.PlatformService.CreatePlatform(r.Context(), p)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, created)
	}
}

// DeletePlatform handles DELETE /api/v1/platform/{id}. An existence check runs
// before deletion; not-found returns 403 rather than 404 per NestJS parity.
func DeletePlatform(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PlatformService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		_, err := deps.PlatformService.GetPlatform(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Legacy parity: delete missing platform returns forbidden.
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.PlatformService.DeletePlatform(r.Context(), id); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// UpdatePlatform handles PUT /api/v1/platform/{id}. A missing platform returns
// 400 Bad Request per NestJS parity (not the more-intuitive 404).
func UpdatePlatform(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PlatformService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		var body struct {
			Name *string `json:"name"`
			Url  string  `json:"url"`
		}
		if !Bind(w, r, &body) {
			return
		}

		p := model.Platform{
			ID:   id,
			Name: body.Name,
			Url:  strings.TrimSpace(body.Url),
		}

		updated, err := deps.PlatformService.UpdatePlatform(r.Context(), p)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Legacy parity: update missing platform returns bad request.
				Error(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}

// GetPlatforms handles GET /api/v1/platforms and returns the list wrapped in a
// "platforms" key, matching the public endpoint response envelope.
func GetPlatforms(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PlatformService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		platforms, err := deps.PlatformService.GetPlatforms(r.Context())
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if platforms == nil {
			platforms = []model.Platform{}
		}
		// Legacy parity: /platforms wraps in { platforms: [...] }
		JSON(w, http.StatusOK, map[string]any{"platforms": platforms})
	}
}
