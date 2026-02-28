package handler

import "net/http"

// --- Group: platform (CRUD with account count) ---

func HandleGetPlatform(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

func HandleCreatePlatform(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeletePlatform(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleUpdatePlatform(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = id   // TODO: pass to service
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// --- Group: platforms (read-only list) ---

func HandleGetPlatforms(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}
