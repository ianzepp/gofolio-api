// Package handler — tag.go implements handlers for /api/v1/tags.
//
// Tags are user-defined labels that can be attached to portfolio holdings.
// The permission model for CreateTag is unusually complex: the NestJS controller
// performs in-controller permission checks rather than relying solely on route
// guards. Specifically, a user with only the createOwnTag permission may create
// a tag scoped to themselves, but must supply their own userId in the body —
// any mismatch returns 400 (not 403) to match the original behaviour. This
// logic is reproduced verbatim in CreateTag below.
//
// Both DeleteTag and UpdateTag return 403 on not-found (instead of 404) to
// avoid leaking whether a tag ID exists. This is intentional NestJS behaviour.
//
// Parity reference: apps/api/src/app/endpoints/tags/tags.controller.ts
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/tags/tags.controller.ts

Routes + request/response shape:
- GET /api/v1/tags (GetTags)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readTags
  Response: TagWithActivityCount[]
- POST /api/v1/tags (CreateTag)
  Request: body data: CreateTagDto { id?: string; name: string; userId?: string }
  Auth/permission: jwt / -
  Response: Tag
- DELETE /api/v1/tags/{id} (DeleteTag)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deleteTag
  Response: Tag (deleted)
- PUT /api/v1/tags/{id} (UpdateTag)
  Request: path id: string (arg: 'id'); body data: UpdateTagDto { name: string }
  Auth/permission: jwt / permissions.updateTag
  Response: Tag
*/

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetTags handles GET /api/v1/tags and returns all tags with their associated
// activity counts. Tags are global (not user-scoped) in the admin list view.
func GetTags(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Legacy behavior note:
		// - NestJS protects this route with AuthGuard('jwt') + readTags permission.
		// - Current Go routing applies auth/permission middleware at registration time.
		if deps == nil || deps.TagService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		tags, err := deps.TagService.GetTagsWithActivityCount(r.Context())
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if tags == nil {
			tags = []model.TagWithActivityCount{}
		}
		JSON(w, http.StatusOK, tags)
	}
}

// CreateTag handles POST /api/v1/tags with in-handler permission logic that
// replicates the NestJS controller's custom guard bypass.
//
// Permission rules:
//   - createTag: may create any tag with any userId.
//   - createOwnTag only: userId in the body must exactly match the caller's ID;
//     a mismatch returns 400 (not 403) to match NestJS behaviour.
//   - Neither permission: 403.
//
// WHY: The NestJS original intentionally skips HasPermissionGuard on this route
// and does the check manually to support the "own tag" nuance. Replicating it
// here keeps the Go port behaviourally identical without custom middleware.
func CreateTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Legacy behavior note:
		// - NestJS intentionally does NOT use HasPermissionGuard here. It performs
		//   custom permission logic in-controller:
		//   createOwnTag OR createTag must be present.
		// - If caller only has createOwnTag, userId must match request.user.id.
		// - That mismatch returns 400 (not 403).
		// Keep this context in mind when adjusting middleware for parity.
		if deps == nil || deps.TagService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		var body struct {
			ID     *string `json:"id"`
			Name   string  `json:"name"`
			UserID *string `json:"userId"`
		}
		if !Bind(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			Error(w, http.StatusBadRequest, "name must be a string")
			return
		}

		// Mirror legacy NestJS in-controller permission checks:
		// - require createOwnTag OR createTag
		// - if only createOwnTag, userId must exactly equal request.user.id
		info := auth.GetAuthInfo(r.Context())
		if info == nil || info.User == nil {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}
		canCreateOwnTag := auth.HasPermission(info.User.Role, auth.CreateOwnTag)
		canCreateTag := auth.HasPermission(info.User.Role, auth.CreateTag)
		if !canCreateOwnTag && !canCreateTag {
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}
		if canCreateOwnTag && !canCreateTag {
			if body.UserID == nil || strings.TrimSpace(*body.UserID) != info.User.ID {
				Error(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
				return
			}
		}

		id := uuid.New().String()
		if body.ID != nil && strings.TrimSpace(*body.ID) != "" {
			id = strings.TrimSpace(*body.ID)
		}

		tag := model.Tag{
			ID:     id,
			Name:   body.Name,
			UserId: body.UserID,
		}

		created, err := deps.TagService.CreateTag(r.Context(), tag)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, created)
	}
}

// DeleteTag handles DELETE /api/v1/tags/{id}. A pre-flight existence check is
// performed before deletion; a missing tag returns 403 (not 404) to match NestJS.
func DeleteTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Legacy behavior note:
		// - If tag does not exist, NestJS returns 403 Forbidden (not 404).
		// This preserves an intentionally non-REST status-code contract.
		if deps == nil || deps.TagService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		if _, err := deps.TagService.GetTag(r.Context(), id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		deleted, err := deps.TagService.DeleteTag(r.Context(), id)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, deleted)
	}
}

// UpdateTag handles PUT /api/v1/tags/{id}. Both id and name must be non-empty
// strings in the body (in addition to the path id). A missing tag returns 403.
func UpdateTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Legacy behavior note:
		// - Request body requires both id and name string validation here.
		// - If target tag does not exist, NestJS returns 403 Forbidden (not 404).
		if deps == nil || deps.TagService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		var body struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if !Bind(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.ID) == "" {
			Error(w, http.StatusBadRequest, "id must be a string")
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			Error(w, http.StatusBadRequest, "name must be a string")
			return
		}

		if _, err := deps.TagService.GetTag(r.Context(), id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		updated, err := deps.TagService.UpdateTag(r.Context(), id, body.Name)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}
