package handler

import "net/http"

func HandleCreateApiKey(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}
