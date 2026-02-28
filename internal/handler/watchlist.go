package handler

import (
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
)

// GetWatchlist returns the user's watched symbols. Supports impersonation.
func GetWatchlist(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if impID := Impersonation(r); impID != "" {
			userID = impID
		}

		items, err := deps.WatchlistService.GetWatchlist(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, items)
	}
}

// CreateWatchlistItem adds a symbol to the user's watchlist.
func CreateWatchlistItem(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			DataSource string `json:"dataSource"`
			Symbol     string `json:"symbol"`
		}
		if !Bind(w, r, &body) {
			return
		}

		if err := deps.WatchlistService.AddWatchlistItem(r.Context(), userID, body.DataSource, body.Symbol); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusCreated, map[string]any{})
	}
}

// DeleteWatchlistItem removes a symbol from the user's watchlist.
func DeleteWatchlistItem(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		if err := deps.WatchlistService.RemoveWatchlistItem(r.Context(), userID, dataSource, symbol); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
}
