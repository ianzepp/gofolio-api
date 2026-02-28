package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/auth-device/auth-device.controller.ts

Routes + request/response shape:
- DELETE /api/v1/auth-device/{id} (HandleDeleteAuthDevice)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deleteAuthDevice
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

// DELETE /auth-device/:id
func HandleDeleteAuthDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
