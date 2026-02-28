package handler

import "net/http"

func HandleGetAiPrompt(w http.ResponseWriter, r *http.Request) {
	mode := r.PathValue("mode")
	_ = mode
	JSON(w, http.StatusOK, map[string]any{})
}
