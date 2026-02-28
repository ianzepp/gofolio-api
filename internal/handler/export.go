package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/export/export.controller.ts

Routes + request/response shape:
- GET /api/v1/export (HandleGetExport)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetExport(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"meta":       map[string]any{},
		"accounts":   []any{},
		"activities": []any{},
	})
}
