package handler

import (
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
)

func GetExport(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		if deps.ExportService == nil {
			JSON(w, http.StatusOK, map[string]any{
				"meta":       map[string]any{},
				"accounts":   []any{},
				"activities": []any{},
			})
			return
		}

		filters := ParseFilters(r)
		resp, err := deps.ExportService.GetExport(r.Context(), userID, filters.Accounts)
		if err != nil {
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}
