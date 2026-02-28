package handler

import "net/http"

func HandleGetWatchlist(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

func HandleCreateWatchlistItem(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeleteWatchlistItem(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource // TODO: pass to service
	_ = symbol     // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
