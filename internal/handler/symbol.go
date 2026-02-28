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

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
)

// GetSymbol returns a closure that handles GET /api/v1/symbol/{dataSource}/{symbol}.
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

// GetSymbolByDate returns a closure that handles GET /api/v1/symbol/{dataSource}/{symbol}/{dateString}.
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

// SymbolLookup returns a closure that handles GET /api/v1/symbol/lookup.
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
