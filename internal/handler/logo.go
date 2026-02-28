package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/logo/logo.controller.ts

Routes + request/response shape:
- GET /api/v1/logo (HandleGetLogo)
  Request: query url: string (arg: 'url')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/logo/{dataSource}/{symbol} (HandleGetLogoBySymbol)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
}

func HandleGetLogoBySymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
}
