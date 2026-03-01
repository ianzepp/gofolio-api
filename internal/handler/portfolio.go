package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/service"
)

var errResolveUserUnauthorized = errors.New("unauthorized")
var errResolveUserForbidden = errors.New("forbidden")

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
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}
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
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}
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
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}

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
func UpdatePortfolioHoldingTags(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}

		var body struct {
			Tags json.RawMessage `json:"tags"`
		}
		if !Bind(w, r, &body) {
			return
		}

		tagIDs, err := parseTagIDs(body.Tags)
		if err != nil {
			Error(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}

		if deps.OrderService == nil || deps.PortfolioService == nil {
			Error(w, http.StatusInternalServerError, "Service unavailable")
			return
		}

		holding, err := deps.PortfolioService.GetHolding(r.Context(), userID, dataSource, symbol)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if holding == nil {
			Error(w, http.StatusNotFound, "Not Found")
			return
		}

		if err := deps.OrderService.UpdateTagsForHolding(r.Context(), userID, dataSource, symbol, tagIDs); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
}

// GET /portfolio/holdings
func GetPortfolioHoldings(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}
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
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}
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
		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}

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
func resolveUserID(r *http.Request, deps *Deps) (string, error) {
	userID := auth.UserID(r)
	if userID == "" {
		return "", errResolveUserUnauthorized
	}

	impersonationID := Impersonation(r)
	if impersonationID == "" {
		return userID, nil
	}

	if deps == nil || deps.UserService == nil {
		return "", errResolveUserForbidden
	}
	user, err := deps.UserService.GetUser(r.Context(), userID)
	if err != nil {
		return "", errResolveUserForbidden
	}
	if !auth.HasPermission(user.Role, auth.ImpersonateAllUsers) {
		return "", errResolveUserForbidden
	}

	return impersonationID, nil
}

func writeResolveUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errResolveUserUnauthorized):
		Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
	case errors.Is(err, errResolveUserForbidden):
		Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
	default:
		Error(w, http.StatusInternalServerError, err.Error())
	}
}

func parseTagIDs(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}, nil
	}

	var ids []string
	if err := json.Unmarshal(raw, &ids); err == nil {
		return ids, nil
	}

	var tags []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &tags); err != nil {
		return nil, err
	}

	ids = make([]string, 0, len(tags))
	for _, t := range tags {
		if t.ID != "" {
			ids = append(ids, t.ID)
		}
	}
	return ids, nil
}
