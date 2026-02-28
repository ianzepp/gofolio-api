package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/market-data/market-data.controller.ts

Routes + request/response shape:
- GET /api/v1/market-data/{dataSource}/{symbol} (HandleGetMarketData)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/market-data/{dataSource}/{symbol} (HandleCreateMarketData)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol'); body data: UpdateBulkMarketDataDto
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/market-data/markets (HandleGetMarkets)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readMarketDataOfMarkets
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetMarketData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleCreateMarketData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleGetMarkets(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}
