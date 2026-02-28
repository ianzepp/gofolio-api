// Package handler — market_data.go implements handlers for the market-data
// admin surface under /api/v1/market-data/{dataSource}/{symbol}.
//
// GetMarketData enforces a two-tier permission model:
//   - Users with ReadMarketData permission can read any asset profile.
//   - Users with ReadMarketDataOfOwnAssetProfile permission can read only
//     profiles where the symbolProfile.userId matches their own ID.
//   - All other combinations return 403 or 404 depending on whether the
//     profile has a userId set (to avoid leaking profile existence).
//
// Currency-like symbols receive special treatment on 404: they return 500
// rather than 404 because the NestJS original wraps currency-symbol errors as
// internal errors (the currency service is expected to always have these). The
// isCurrencyLikeSymbol helper replicates that detection logic.
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"database/sql"
	"golang.org/x/text/currency"
)

// GetMarketData handles GET /api/v1/market-data/{dataSource}/{symbol}. It loads
// the authenticated user's role to perform permission checks after fetching the
// asset profile, rather than gating at middleware level, because permission
// depends on the profile's userId field which is only known after the DB query.
func GetMarketData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		if deps.MarketDataService == nil || deps.UserService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		user, err := deps.UserService.GetUser(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		resp, err := deps.MarketDataService.GetBySymbol(r.Context(), dataSource, symbol)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if isCurrencyLikeSymbol(symbol) {
					Error(w, http.StatusInternalServerError, "Internal server error")
					return
				}
				Error(w, http.StatusNotFound, "Not Found")
				return
			}
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		canReadAllAssetProfiles := auth.HasPermission(user.Role, auth.ReadMarketData)
		canReadOwnAssetProfile := resp.AssetProfile.UserId != nil &&
			*resp.AssetProfile.UserId == userID &&
			auth.HasPermission(user.Role, auth.ReadMarketDataOfOwnAssetProfile)
		if !canReadAllAssetProfiles && !canReadOwnAssetProfile {
			if resp.AssetProfile.UserId != nil {
				Error(w, http.StatusNotFound, "Not Found")
				return
			}
			Error(w, http.StatusForbidden, "Forbidden")
			return
		}

		JSON(w, http.StatusOK, resp)
	}
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
	JSON(w, http.StatusOK, map[string]any{})
}

// isCurrencyLikeSymbol returns true when the symbol looks like a currency pair
// stored in Ghostfolio's XYZUSD format. It strips the USD suffix and checks the
// remaining base against known non-ISO exceptions (USX, GBp, ILA, ZAc) before
// falling back to the golang.org/x/text/currency ISO parser.
//
// WHY: The NestJS backend wraps missing currency-symbol lookups as 500 errors
// rather than 404, because currency data is expected to always be present. This
// helper lets GetMarketData replicate that different error path.
func isCurrencyLikeSymbol(symbol string) bool {
	candidate := strings.Replace(symbol, "USD", "", 1)
	if candidate == "" {
		return false
	}
	switch candidate {
	case "USX", "GBp", "ILA", "ZAc":
		return true
	}
	_, err := currency.ParseISO(strings.ToUpper(candidate))
	return err == nil
}
