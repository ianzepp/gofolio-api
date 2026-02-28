package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/asset/asset.controller.ts

Routes + request/response shape:
- GET /api/v1/asset/{dataSource}/{symbol} (HandleGetAsset)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetAsset(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}
