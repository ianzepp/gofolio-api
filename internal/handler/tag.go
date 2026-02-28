package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/tags/tags.controller.ts

Routes + request/response shape:
- GET /api/v1/tags (HandleGetTags)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readTags
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/tags (HandleCreateTag)
  Request: body data: CreateTagDto
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/tags/{id} (HandleDeleteTag)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deleteTag
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- PUT /api/v1/tags/{id} (HandleUpdateTag)
  Request: path id: string (arg: 'id'); body data: UpdateTagDto
  Auth/permission: jwt / permissions.updateTag
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetTags(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

func HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = id   // TODO: pass to service
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
