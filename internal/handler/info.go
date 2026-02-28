package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/info/info.controller.ts

Routes + request/response shape:
- GET /api/v1/info (HandleGetInfo)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	// Legacy behavior note:
	// Frontend boot expects this payload shape (baseCurrency/currencies/
	// globalPermissions/statistics keys), even when values are empty.
	JSON(w, http.StatusOK, map[string]any{
		"baseCurrency":      "USD",
		"benchmarks":        []any{},
		"currencies":        []string{"USD"},
		"demoAuthToken":     "",
		"globalPermissions": []string{},
		"statistics":        nil,
	})
}
