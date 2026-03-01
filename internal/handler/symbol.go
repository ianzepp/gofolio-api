// Package handler — symbol.go implements handlers for /api/v1/symbol/*.
//
// Symbols are identified by a (dataSource, symbol) pair. The dataSource path
// segment is validated against the known set of data source identifiers; unknown
// values return 404 so clients receive a clear signal rather than a downstream
// provider error.
//
// GetSymbol optionally includes historical data when the includeHistoricalData
// query parameter is set to a positive integer (days). Values ≤ 0 are clamped
// to 0 (no historical data) to prevent negative-day queries from reaching the
// service layer.
//
// Parity reference: apps/api/src/app/symbol/symbol.controller.ts
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/symbol/symbol.controller.ts

Routes + request/response shape:
- GET /api/v1/symbol/{dataSource}/{symbol} (GetSymbol)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: public / -
- GET /api/v1/symbol/{dataSource}/{symbol}/{dateString} (GetSymbolByDate)
  Request: path dataSource: DataSource (arg: 'dataSource'); dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / -
- GET /api/v1/symbol/lookup (SymbolLookup)
  Request: query: query (string)
  Auth/permission: jwt / -
*/

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/service"
)

// GetSymbol handles GET /api/v1/symbol/{dataSource}/{symbol}. The optional
// includeHistoricalData query parameter specifies how many trailing days of
// price history to include; it is clamped to ≥ 0 before passing to the service.
func GetSymbol(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.SymbolService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		dataSource := model.DataSource(r.PathValue("dataSource"))
		symbol := r.PathValue("symbol")

		if !isValidDataSource(dataSource) {
			Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}

		includeHistoricalDays := 0
		if includeHistoricalRaw := r.URL.Query().Get("includeHistoricalData"); includeHistoricalRaw != "" {
			includeHistoricalDays, _ = strconv.Atoi(includeHistoricalRaw)
			if includeHistoricalDays < 0 {
				includeHistoricalDays = 0
			}
		}

		item, err := deps.SymbolService.Get(r.Context(), dataSource, symbol, includeHistoricalDays)
		if err != nil {
			if errors.Is(err, service.ErrSymbolNotFound) {
				Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, item)
	}
}

// GetSymbolByDate handles GET /api/v1/symbol/{dataSource}/{symbol}/{dateString}
// and returns the market price for the symbol on a specific date. Date must be
// in YYYY-MM-DD format; an invalid format returns 400.
func GetSymbolByDate(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.SymbolService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		dataSource := model.DataSource(r.PathValue("dataSource"))
		symbol := r.PathValue("symbol")
		dateString := r.PathValue("dateString")
		if !isValidDataSource(dataSource) {
			Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}

		date, err := time.Parse("2006-01-02", dateString)
		if err != nil {
			Error(w, http.StatusBadRequest, "Invalid date format, expected YYYY-MM-DD")
			return
		}

		result, err := deps.SymbolService.GetForDate(r.Context(), dataSource, symbol, date)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, result)
	}
}

// isValidDataSource returns true if dataSource is one of the known DataSource
// enum values. This guard prevents routing arbitrary strings to provider
// implementations that might misinterpret them.
func isValidDataSource(dataSource model.DataSource) bool {
	switch dataSource {
	case model.DataSourceAlphaVantage,
		model.DataSourceCoinGecko,
		model.DataSourceEODHistoricalData,
		model.DataSourceFinancialModelingPrep,
		model.DataSourceGhostfolio,
		model.DataSourceGoogleSheets,
		model.DataSourceManual,
		model.DataSourceRapidAPI,
		model.DataSourceYahoo:
		return true
	default:
		return false
	}
}

// SymbolLookup handles GET /api/v1/symbol/lookup. An empty query returns an
// empty items list without hitting the provider, matching the NestJS guard.
func SymbolLookup(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.SymbolService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		query := r.URL.Query().Get("query")
		if query == "" {
			JSON(w, http.StatusOK, map[string]any{"items": []any{}})
			return
		}

		includeIndices := r.URL.Query().Get("includeIndices") == "true"

		result, err := deps.SymbolService.Lookup(r.Context(), query, includeIndices)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, result)
	}
}
