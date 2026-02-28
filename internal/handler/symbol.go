package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/symbol/symbol.controller.ts

Routes + request/response shape:
- GET /api/v1/symbol/{dataSource}/{symbol} (HandleGetSymbol)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/symbol/{dataSource}/{symbol}/{dateString} (HandleGetSymbolByDate)
  Request: path dataSource: DataSource (arg: 'dataSource'); dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/symbol/lookup (HandleSymbolLookup)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetSymbol(w http.ResponseWriter, r *http.Request) {
	// Legacy behavior note:
	// Current NestJS contract keeps this endpoint public, while lookup/date
	// variants are authenticated.
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetSymbolByDate(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	dateString := r.PathValue("dateString")
	_ = dataSource
	_ = symbol
	_ = dateString
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleSymbolLookup(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	_ = filters
	JSON(w, http.StatusOK, map[string]any{"items": []any{}})
}
