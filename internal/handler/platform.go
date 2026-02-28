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
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
)

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

func DeletePlatform(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PlatformService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		if err := deps.PlatformService.DeletePlatform(r.Context(), id); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

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
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}

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
