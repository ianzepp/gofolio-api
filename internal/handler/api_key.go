package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/api-keys/api-keys.controller.ts

Routes + request/response shape:
- POST /api/v1/api-keys (HandleCreateApiKey)
  Request: no explicit request params
  Auth/permission: jwt / permissions.createApiKey
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleCreateApiKey(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}
