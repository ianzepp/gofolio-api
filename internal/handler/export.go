package handler

import "net/http"

func HandleGetExport(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"meta":       map[string]any{},
		"accounts":   []any{},
		"activities": []any{},
	})
}
