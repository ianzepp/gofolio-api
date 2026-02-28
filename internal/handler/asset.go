package handler

import (
	"database/sql"
	"errors"
	"net/http"
)

func GetAsset(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		if deps.AssetService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		resp, err := deps.AssetService.GetAsset(r.Context(), dataSource, symbol)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusNotFound, "Not Found")
				return
			}
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}
