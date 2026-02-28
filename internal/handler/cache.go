package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/cache/cache.controller.ts

Routes + request/response shape:
- POST /api/v1/cache/flush (HandleFlushCache)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

// POST /cache/flush
func HandleFlushCache(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}
