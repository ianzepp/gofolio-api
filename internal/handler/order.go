package handler

import "net/http"

// DELETE /order (bulk delete)
func HandleDeleteOrders(w http.ResponseWriter, r *http.Request) {
	impersonationId := Impersonation(r)
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /order
func HandleGetOrders(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	page := ParsePagination(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = page
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{"activities": []any{}})
}

// POST /order
func HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

// DELETE /order/:id
func HandleDeleteOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /order/:id
func HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}

// PUT /order/:id
func HandleUpdateOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = id
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}
