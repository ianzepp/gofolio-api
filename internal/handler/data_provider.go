package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/data-providers/ghostfolio/ghostfolio.controller.ts

Routes + request/response shape:
- GET /api/v1/data-providers/ghostfolio/asset-profile/{symbol} (HandleGetDataProviderAssetProfile)
  Request: path symbol: string (arg: 'symbol')
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/data-providers/ghostfolio/dividends/{symbol} (HandleGetDataProviderDividends)
  Request: path symbol: string (arg: 'symbol'); query query: GetDividendsDto
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/data-providers/ghostfolio/historical/{symbol} (HandleGetDataProviderHistorical)
  Request: path symbol: string (arg: 'symbol'); query query: GetHistoricalDto
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/data-providers/ghostfolio/lookup (HandleDataProviderLookup)
  Request: no explicit request params
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/data-providers/ghostfolio/quotes (HandleGetDataProviderQuotes)
  Request: query query: GetQuotesDto
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/data-providers/ghostfolio/status (HandleGetDataProviderStatus)
  Request: no explicit request params
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetDataProviderAssetProfile(w http.ResponseWriter, r *http.Request) {
	// Legacy behavior note:
	// Ghostfolio data-provider endpoints use API-key auth (not jwt) in NestJS.
	symbol := r.PathValue("symbol")
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetDataProviderDividends(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{"dividends": []any{}})
}

func HandleGetDataProviderHistorical(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{"historicalData": []any{}})
}

func HandleDataProviderLookup(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"items": []any{}})
}

func HandleGetDataProviderQuotes(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"quotes": []any{}})
}

func HandleGetDataProviderStatus(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}
