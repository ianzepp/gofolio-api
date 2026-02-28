package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/jackc/pgx/v5"
	"golang.org/x/text/currency"
)

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
			if errors.Is(err, pgx.ErrNoRows) {
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
