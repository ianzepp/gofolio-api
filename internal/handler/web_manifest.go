package handler

import "net/http"

func HandleGetWebManifest(w http.ResponseWriter, r *http.Request) {
	languageCode := r.PathValue("languageCode")
	_ = languageCode
	JSON(w, http.StatusOK, map[string]any{})
}
