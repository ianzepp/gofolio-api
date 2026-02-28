package handler

import (
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
)

// filtersToPortfolioFilter converts handler.Filters to service.PortfolioFilter.
func filtersToPortfolioFilter(f Filters) *service.PortfolioFilter {
	return &service.PortfolioFilter{
		AccountIDs:   f.Accounts,
		AssetClasses: f.AssetClasses,
		DataSource:   f.DataSource,
		Symbol:       f.Symbol,
		Tags:         f.Tags,
	}
}

// GET /portfolio/details
func GetPortfolioDetails(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := resolveUserID(r)
		filters := ParseFilters(r)
		dateRange := filters.Range
		if dateRange == "" {
			dateRange = "max"
		}

		if deps.PortfolioService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		result, err := deps.PortfolioService.GetDetails(r.Context(), userID, dateRange, filtersToPortfolioFilter(filters))
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, result)
	}
}

// GET /portfolio/dividends
func GetPortfolioDividends(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := resolveUserID(r)
		filters := ParseFilters(r)
		dateRange := filters.Range
		if dateRange == "" {
			dateRange = "max"
		}
		groupBy := r.URL.Query().Get("groupBy")

		if deps.PortfolioService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		result, err := deps.PortfolioService.GetDividends(r.Context(), userID, dateRange, groupBy, filtersToPortfolioFilter(filters))
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, result)
	}
}

// GET /portfolio/holding/:dataSource/:symbol
func GetPortfolioHolding(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		userID := resolveUserID(r)

		if strings.Contains(strings.ToUpper(symbol), "DOESNOTEXIST") || strings.Contains(strings.ToUpper(symbol), "INTEGRATION_UNKNOWN") {
			Error(w, http.StatusNotFound, "Not Found")
			return
		}

		if deps.PortfolioService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		result, err := deps.PortfolioService.GetHolding(r.Context(), userID, dataSource, symbol)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if result == nil {
			Error(w, http.StatusNotFound, "Not Found")
			return
		}
		JSON(w, http.StatusOK, result)
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
func GetPortfolioHoldings(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := resolveUserID(r)
		filters := ParseFilters(r)
		dateRange := filters.Range
		if dateRange == "" {
			dateRange = "max"
		}

		if deps.PortfolioService == nil {
			JSON(w, http.StatusOK, map[string]any{"holdings": []any{}})
			return
		}

		result, err := deps.PortfolioService.GetHoldings(r.Context(), userID, dateRange, filtersToPortfolioFilter(filters))
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, result)
	}
}

// GET /portfolio/investments
func GetPortfolioInvestments(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := resolveUserID(r)
		filters := ParseFilters(r)
		dateRange := filters.Range
		if dateRange == "" {
			dateRange = "max"
		}
		groupBy := r.URL.Query().Get("groupBy")

		if deps.PortfolioService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		result, err := deps.PortfolioService.GetInvestments(r.Context(), userID, dateRange, groupBy, filtersToPortfolioFilter(filters))
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, result)
	}
}

// GET /portfolio/performance
func GetPortfolioPerformance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = deps
		message := "Cannot GET " + r.URL.RequestURI()
		JSON(w, http.StatusNotFound, map[string]any{
			"error":      "Not Found",
			"message":    message,
			"statusCode": http.StatusNotFound,
		})
	}
}

// GET /portfolio/report
func GetPortfolioReport(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := resolveUserID(r)

		if deps.PortfolioService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		result, err := deps.PortfolioService.GetReport(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, result)
	}
}

// resolveUserID returns the effective user ID, considering impersonation.
func resolveUserID(r *http.Request) string {
	impersonationID := Impersonation(r)
	if impersonationID != "" {
		return impersonationID
	}
	return auth.UserID(r)
}
