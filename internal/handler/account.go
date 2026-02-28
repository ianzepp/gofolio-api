package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/account/account.controller.ts

Routes + request/response shape:
- GET /api/v1/account (GetAccounts)
  Request: no explicit request params
  Auth/permission: jwt / -
  Response: { accounts: Account[], totalBalanceInBaseCurrency: number, ... }
- POST /api/v1/account (CreateAccount)
  Request: body data: CreateAccountDto
  Auth/permission: jwt / permissions.createAccount
  Response: Account
- DELETE /api/v1/account/{id} (DeleteAccount)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.deleteAccount
  Response: Account
- GET /api/v1/account/{id} (GetAccount)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / -
  Response: Account
- PUT /api/v1/account/{id} (UpdateAccount)
  Request: path id: string (arg: 'id'); body data: UpdateAccountDto
  Auth/permission: jwt / permissions.updateAccount
  Response: Account
- GET /api/v1/account/{id}/balances (GetAccountBalances)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / -
  Response: { balances: AccountBalance[] }
- POST /api/v1/account/transfer-balance (TransferBalance)
  Request: body data: TransferBalanceDto
  Auth/permission: jwt / permissions.updateAccount
  Response: void (HTTP 200)
*/

import (
	"net/http"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
	"github.com/google/uuid"
)

func GetAccounts(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		// Support impersonation
		if impID := Impersonation(r); impID != "" {
			userID = impID
		}

		accounts, err := deps.AccountService.GetAccounts(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if accounts == nil {
			accounts = []model.Account{}
		}

		totalBalance := 0.0
		for _, account := range accounts {
			totalBalance += account.Balance
		}

		// Legacy parity: include aggregate keys expected by client/API contract.
		JSON(w, http.StatusOK, map[string]any{
			"accounts":                      accounts,
			"activitiesCount":               0,
			"totalBalanceInBaseCurrency":    totalBalance,
			"totalDividendInBaseCurrency":   0,
			"totalInterestInBaseCurrency":   0,
			"totalValueInBaseCurrency":      totalBalance,
		})
	}
}

func CreateAccount(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			Balance    float64 `json:"balance"`
			Comment    *string `json:"comment"`
			Currency   *string `json:"currency"`
			ID         *string `json:"id"`
			IsExcluded bool    `json:"isExcluded"`
			Name       *string `json:"name"`
			PlatformId *string `json:"platformId"`
		}
		if !Bind(w, r, &body) {
			return
		}

		id := uuid.New().String()
		if body.ID != nil && strings.TrimSpace(*body.ID) != "" {
			id = strings.TrimSpace(*body.ID)
		}

		acct := model.Account{
			ID:         id,
			UserId:     userID,
			Balance:    body.Balance,
			Comment:    body.Comment,
			Currency:   body.Currency,
			IsExcluded: body.IsExcluded,
			Name:       body.Name,
			PlatformId: body.PlatformId,
		}

		created, err := deps.AccountService.CreateAccount(r.Context(), acct)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, created)
	}
}

func DeleteAccount(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		if err := deps.AccountService.DeleteAccount(r.Context(), id, userID); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

func GetAccount(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		if impID := Impersonation(r); impID != "" {
			userID = impID
		}

		id := r.PathValue("id")
		acct, err := deps.AccountService.GetAccount(r.Context(), id, userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, acct)
	}
}

func UpdateAccount(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		var body struct {
			Balance    float64 `json:"balance"`
			Comment    *string `json:"comment"`
			Currency   *string `json:"currency"`
			IsExcluded bool    `json:"isExcluded"`
			Name       *string `json:"name"`
			PlatformId *string `json:"platformId"`
		}
		if !Bind(w, r, &body) {
			return
		}

		acct := model.Account{
			ID:         id,
			UserId:     userID,
			Balance:    body.Balance,
			Comment:    body.Comment,
			Currency:   body.Currency,
			IsExcluded: body.IsExcluded,
			Name:       body.Name,
			PlatformId: body.PlatformId,
		}

		updated, err := deps.AccountService.UpdateAccount(r.Context(), acct)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}

func GetAccountBalances(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		// Legacy parity: returns { balances: [] }
		// AccountBalance queries need a dedicated repo method; stub for now.
		JSON(w, http.StatusOK, map[string]any{"balances": []any{}})
	}
}

func TransferBalance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			AccountIdFrom string  `json:"accountIdFrom"`
			AccountIdTo   string  `json:"accountIdTo"`
			Balance       float64 `json:"balance"`
		}
		if !Bind(w, r, &body) {
			return
		}

		accounts, err := deps.AccountService.GetAccounts(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		var accountFrom *model.Account
		var accountTo *model.Account
		for i := range accounts {
			if accounts[i].ID == body.AccountIdFrom {
				accountFrom = &accounts[i]
			}
			if accounts[i].ID == body.AccountIdTo {
				accountTo = &accounts[i]
			}
		}

		// Legacy parity: account IDs must both belong to the caller.
		if accountFrom == nil || accountTo == nil {
			Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}
		if accountFrom.ID == accountTo.ID {
			Error(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
			return
		}
		if accountFrom.Balance < body.Balance {
			Error(w, http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
			return
		}

		err = deps.AccountService.TransferBalance(r.Context(), service.TransferRequest{
			FromID:     body.AccountIdFrom,
			FromUserID: userID,
			ToID:       body.AccountIdTo,
			ToUserID:   userID,
			Amount:     body.Balance,
		})
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Legacy parity: transfer returns void (HTTP 200 with no body)
		w.WriteHeader(http.StatusOK)
	}
}
