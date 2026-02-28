package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/account-balance/account-balance.controller.ts

Routes + request/response shape:
- POST /api/v1/account-balance (HandleCreateAccountBalance)
  Request: body data: CreateAccountBalanceDto
  Auth/permission: jwt / permissions.createAccountBalance
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/account-balance/{id} (HandleDeleteAccountBalance)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deleteAccountBalance
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

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
