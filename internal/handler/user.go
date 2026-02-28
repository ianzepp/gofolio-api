package handler

import "net/http"

// DELETE /user
func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /user
func HandleGetUser(w http.ResponseWriter, r *http.Request) {
	lang := r.Header.Get("Accept-Language")
	_ = lang // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /user
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}

// DELETE /user/:id
func HandleDeleteUserById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /user/:id/access-token
func HandleCreateUserAccessTokenById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

// POST /user/access-token
func HandleCreateUserAccessToken(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusCreated, map[string]any{})
}

// PUT /user/setting
func HandleUpdateUserSetting(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
