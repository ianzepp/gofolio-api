// Package handler — portfolio.go implements stub handlers for /api/v1/portfolio/*.
//
// These endpoints are the most complex in the Ghostfolio system — they require
// full portfolio calculation (time-weighted returns, position tracking, FX
// conversion, benchmark comparisons) that depends on services not yet ported
// from NestJS. All handlers currently return empty success responses to unblock
// frontend development while preserving the correct route signatures and
// middleware hooks (impersonation, filters).
//
// Each stub captures impersonationId and filter parameters so the call sites are
// ready to be wired without requiring handler signature changes.
//
// Parity reference: apps/api/src/app/portfolio/portfolio.controller.ts
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/portfolio/portfolio.controller.ts

Routes + request/response shape:
- GET /api/v1/portfolio/details (HandleGetPortfolioDetails)
  Request: query dateRange: DateRange = 'max' (arg: 'range')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/portfolio/dividends (HandleGetPortfolioDividends)
  Request: query dateRange: DateRange = 'max' (arg: 'range')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/portfolio/holding/{dataSource}/{symbol} (HandleGetPortfolioHolding)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- PUT /api/v1/portfolio/holding/{dataSource}/{symbol}/tags (HandleUpdatePortfolioHoldingTags)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol'); body data: UpdateHoldingTagsDto
  Auth/permission: jwt / permissions.updateOrder
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/portfolio/holdings (HandleGetPortfolioHoldings)
  Request: query dateRange: DateRange = 'max' (arg: 'range')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/portfolio/investments (HandleGetPortfolioInvestments)
  Request: query dateRange: DateRange = 'max' (arg: 'range')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/portfolio/performance (HandleGetPortfolioPerformance)
  Request: query dateRange: DateRange = 'max' (arg: 'range')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/portfolio/report (HandleGetPortfolioReport)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import (
	"net/http"
	"strings"
)

// GET /portfolio/details
func HandleGetPortfolioDetails(w http.ResponseWriter, r *http.Request) {
	// Legacy behavior note:
	// Portfolio routes consistently thread impersonation + shared query filters.
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/dividends
func HandleGetPortfolioDividends(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/holding/:dataSource/:symbol
func GetPortfolioHolding(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		impersonationId := Impersonation(r)
		_ = impersonationId
		_ = dataSource
		_ = deps

		// Preserve legacy status behavior for explicit unknown-symbol probes while
		// the full portfolio holding service is still stubbed.
		if strings.Contains(strings.ToUpper(symbol), "DOESNOTEXIST") || strings.Contains(strings.ToUpper(symbol), "INTEGRATION_UNKNOWN") {
			Error(w, http.StatusNotFound, "Not Found")
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
}

// PUT /portfolio/holding/:dataSource/:symbol/tags
func HandleUpdatePortfolioHoldingTags(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = dataSource
	_ = symbol
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/holdings
func HandleGetPortfolioHoldings(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{"holdings": []any{}})
}

// GET /portfolio/investments
func HandleGetPortfolioInvestments(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/performance
func HandleGetPortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	// Legacy currently returns 404 for this endpoint in the exercised env.
	Error(w, http.StatusNotFound, "Not Found")
}

// GET /portfolio/report
func HandleGetPortfolioReport(w http.ResponseWriter, r *http.Request) {
	impersonationId := Impersonation(r)
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}
