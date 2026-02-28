package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/watchlist/watchlist.controller.ts

Routes + request/response shape:
- GET /api/v1/watchlist (HandleGetWatchlist)
  Request: no explicit request params
  Auth/permission: jwt / permissions.readWatchlist
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/watchlist (HandleCreateWatchlistItem)
  Request: body data: CreateWatchlistItemDto
  Auth/permission: jwt / permissions.createWatchlistItem
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/watchlist/{dataSource}/{symbol} (HandleDeleteWatchlistItem)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / permissions.deleteWatchlistItem
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetWatchlist(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

func HandleCreateWatchlistItem(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeleteWatchlistItem(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource // TODO: pass to service
	_ = symbol     // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
