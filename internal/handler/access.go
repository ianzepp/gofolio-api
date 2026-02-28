package handler

import "net/http"

// HandleGetAccesses handles GET /access
func HandleGetAccesses(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"accesses": []any{}})
}

// HandleCreateAccess handles POST /access
func HandleCreateAccess(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

// HandleDeleteAccess handles DELETE /access/:id
func HandleDeleteAccess(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}

// HandleUpdateAccess handles PUT /access/:id
func HandleUpdateAccess(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = id
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}
