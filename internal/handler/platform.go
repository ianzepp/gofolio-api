package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/platform/platform.controller.ts
- apps/api/src/app/endpoints/platforms/platforms.controller.ts

Routes + request/response shape:
- GET /api/v1/platform (HandleGetPlatform)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readPlatformsWithAccountCount
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/platform (HandleCreatePlatform)
  Request: body data: CreatePlatformDto
  Auth/permission: jwt / permissions.createPlatform
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/platform/{id} (HandleDeletePlatform)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deletePlatform
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- PUT /api/v1/platform/{id} (HandleUpdatePlatform)
  Request: path id: string (arg: 'id'); body data: UpdatePlatformDto
  Auth/permission: jwt / permissions.updatePlatform
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/platforms (HandleGetPlatforms)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readPlatforms
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

// --- Group: platform (CRUD with account count) ---

func HandleGetPlatform(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

func HandleCreatePlatform(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeletePlatform(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleUpdatePlatform(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = id   // TODO: pass to service
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// --- Group: platforms (read-only list) ---

func HandleGetPlatforms(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}
