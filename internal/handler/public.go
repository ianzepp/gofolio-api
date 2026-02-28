package handler

import "net/http"

func HandleGetPublicPortfolio(w http.ResponseWriter, r *http.Request) {
	accessId := r.PathValue("accessId")
	_ = accessId
	JSON(w, http.StatusOK, map[string]any{})
}
