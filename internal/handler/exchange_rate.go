package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/exchange-rate/exchange-rate.controller.ts

Routes + request/response shape:
- GET /api/v1/exchange-rate/{symbol}/{dateString} (HandleGetExchangeRate)
  Request: path dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetExchangeRate(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	dateString := r.PathValue("dateString")
	_ = symbol
	_ = dateString
	JSON(w, http.StatusOK, map[string]any{})
}
