package handler

import "net/http"

// DELETE /auth-device/:id
func HandleDeleteAuthDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
