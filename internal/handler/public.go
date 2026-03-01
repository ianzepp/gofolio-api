package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"os"

	"github.com/ianzepp/gofolio-api/internal/db"
	"github.com/ianzepp/gofolio-api/internal/repository"
)

// GetPublicPortfolio handles GET /api/v1/public/{accessId}/portfolio.
// This is a public (unauthenticated) endpoint that returns a portfolio
// summary for a shared access link. The access ID maps to an Access record
// whose owner's portfolio is then computed and returned with limited detail.
func GetPublicPortfolio(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccessService == nil || deps.PortfolioService == nil || deps.UserService == nil || deps.DB == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		accessID := r.PathValue("accessId")

		// Look up the access grant
		access, err := deps.AccessService.GetAccess(r.Context(), accessID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
				return
			}
			Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}

		ownerID := access.UserId
		hasDetails := true
		if os.Getenv("ENABLE_FEATURE_SUBSCRIPTION") == "true" {
			subscriptionRepo := repository.NewSubscriptionRepository(deps.DB)
			if _, err := subscriptionRepo.FindActiveByUserID(r.Context(), ownerID); err != nil {
				// db.ErrNoRows is an alias of sql.ErrNoRows in internal/db.
				if errors.Is(err, db.ErrNoRows) {
					hasDetails = false
				} else {
					Error(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
		}

		// Get portfolio details (holdings map)
		details, err := deps.PortfolioService.GetDetails(r.Context(), ownerID, "max", nil)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Get performance for 1d, max, ytd
		perf1d, _ := deps.PortfolioService.GetPerformance(r.Context(), ownerID, "1d", nil)
		perfMax, _ := deps.PortfolioService.GetPerformance(r.Context(), ownerID, "max", nil)
		perfYtd, _ := deps.PortfolioService.GetPerformance(r.Context(), ownerID, "ytd", nil)

		// Build performance response
		performance := map[string]any{
			"1d":  map[string]any{"relativeChange": float64(0)},
			"max": map[string]any{"relativeChange": float64(0)},
			"ytd": map[string]any{"relativeChange": float64(0)},
		}
		if perf1d != nil {
			performance["1d"] = map[string]any{
				"relativeChange": perf1d.Performance.NetPerformancePercentageWithCurrencyEffect,
			}
		}
		if perfMax != nil {
			performance["max"] = map[string]any{
				"relativeChange": perfMax.Performance.NetPerformancePercentageWithCurrencyEffect,
			}
		}
		if perfYtd != nil {
			performance["ytd"] = map[string]any{
				"relativeChange": perfYtd.Performance.NetPerformancePercentageWithCurrencyEffect,
			}
		}

		// Compute total value for allocation percentages
		totalValue := float64(0)
		for _, h := range details.Holdings {
			if h.ValueInBaseCurrency != nil {
				totalValue += *h.ValueInBaseCurrency
			}
		}

		// Build public holdings map
		publicHoldings := make(map[string]any, len(details.Holdings))
		for symbol, h := range details.Holdings {
			allocation := float64(0)
			if totalValue > 0 && h.ValueInBaseCurrency != nil {
				allocation = *h.ValueInBaseCurrency / totalValue
			}

			netPerfPctCE := float64(0)
			if h.NetPerformancePercentWithCurrencyEffect != nil {
				netPerfPctCE = *h.NetPerformancePercentWithCurrencyEffect
			}

			publicHoldings[symbol] = map[string]any{
				"allocationInPercentage": allocation,
				"assetClass":             h.AssetClass,
				"countries":              h.Countries,
				"currency":               h.Currency,
				"dataSource":             h.DataSource,
				"dateOfFirstActivity":    h.DateOfFirstActivity,
				"markets":                nil,
				"name":                   h.Name,
				"netPerformancePercentWithCurrencyEffect": netPerfPctCE,
				"sectors":           h.Sectors,
				"symbol":            h.Symbol,
				"url":               h.URL,
				"valueInPercentage": allocation,
			}
		}

		var alias *string
		if access.Alias != nil {
			alias = access.Alias
		}

		JSON(w, http.StatusOK, map[string]any{
			"alias":            alias,
			"createdAt":        access.CreatedAt,
			"hasDetails":       hasDetails,
			"holdings":         publicHoldings,
			"latestActivities": []any{},
			"markets":          map[string]any{},
			"performance":      performance,
		})
	}
}
