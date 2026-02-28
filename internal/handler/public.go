package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/public/public.controller.ts

Routes + request/response shape:
- GET /api/v1/public/{accessId}/portfolio (HandleGetPublicPortfolio)
  Request: path accessId: string (arg: 'accessId')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetPublicPortfolio(w http.ResponseWriter, r *http.Request) {
	accessId := r.PathValue("accessId")
	_ = accessId
	JSON(w, http.StatusOK, map[string]any{})
}
