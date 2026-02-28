package handler

import "net/http"

// HandleCreateAccountBalance handles POST /account-balance
func HandleCreateAccountBalance(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

// HandleDeleteAccountBalance handles DELETE /account-balance/:id
func HandleDeleteAccountBalance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}
