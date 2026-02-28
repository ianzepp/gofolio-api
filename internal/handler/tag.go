package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/tags/tags.controller.ts

Routes + request/response shape:
- GET /api/v1/tags (GetTags)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readTags
  Response: Tag[]
- POST /api/v1/tags (CreateTag)
  Request: body data: CreateTagDto { name: string }
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
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
)

func GetTags(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tags, err := deps.TagService.GetTags(r.Context())
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if tags == nil {
			tags = []model.Tag{}
		}
		JSON(w, http.StatusOK, tags)
	}
}

func CreateTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if !Bind(w, r, &body) {
			return
		}

		tag := model.Tag{
			ID:   uuid.New().String(),
			Name: body.Name,
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
		id := r.PathValue("id")
		if err := deps.TagService.DeleteTag(r.Context(), id); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

func UpdateTag(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var body struct {
			Name string `json:"name"`
		}
		if !Bind(w, r, &body) {
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
