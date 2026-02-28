package handler

import "net/http"

// POST /cache/flush
func HandleFlushCache(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}
