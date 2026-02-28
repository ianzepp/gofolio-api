package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/assets/assets.controller.ts

Routes + request/response shape:
- GET /assets/{languageCode}/site.webmanifest (HandleGetWebManifest)
  Request: path languageCode: string (arg: 'languageCode')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetWebManifest(w http.ResponseWriter, r *http.Request) {
	languageCode := r.PathValue("languageCode")
	_ = languageCode
	JSON(w, http.StatusOK, map[string]any{})
}
