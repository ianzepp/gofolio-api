package handler

import "net/http"

// HandleGetAccounts handles GET /account
func HandleGetAccounts(w http.ResponseWriter, r *http.Request) {
	impersonationId := Impersonation(r)
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{"accounts": []any{}})
}

// HandleCreateAccount handles POST /account
func HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

// HandleDeleteAccount handles DELETE /account/:id
func HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}

// HandleGetAccount handles GET /account/:id
func HandleGetAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}

// HandleUpdateAccount handles PUT /account/:id
func HandleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = id
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// HandleGetAccountBalances handles GET /account/:id/balances
func HandleGetAccountBalances(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{"balances": []any{}})
}

// HandleTransferBalance handles POST /account/transfer-balance
func HandleTransferBalance(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}
