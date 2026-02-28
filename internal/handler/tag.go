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

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetTags(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func CreateTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func DeleteTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func UpdateTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
