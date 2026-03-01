// Package handler — account.go implements handlers for /api/v1/account.
//
// Accounts represent brokerage/bank accounts that hold cash and/or investment
// positions. The GetAccounts response is deliberately aggregated: it rolls up
// activity counts and portfolio value metrics that the NestJS client expects on
// the accounts list screen, avoiding a separate analytics round-trip.
//
// Impersonation is supported on read endpoints: when an impersonation header is
// present the handler substitutes the target user's ID, enabling admin viewing
// of another user's accounts without the target's JWT.
//
// Parity reference: apps/api/src/app/account/account.controller.ts
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

	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/service"
	"github.com/google/uuid"
)

// GetAccounts handles GET /api/v1/account and returns the user's accounts along
// with aggregated portfolio metrics. The metrics (activitiesCount, totalBalance,
// totalDividend, totalInterest, totalValue) are computed in-handler by iterating
// all orders because neither the account nor the order service currently exposes
// an aggregate query.
//
// WHY: The NestJS controller performs this aggregation in-process as well. Until
// portfolio calculation services are ported, the inline approach preserves exact
// API parity without introducing a partial service abstraction.
//
// EDGE: If the order service is unavailable the accounts list is still returned;
// activity-derived metrics will be zero rather than an error.
func GetAccounts(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
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

		activitiesCount := 0
		totalDividend := 0.0
		totalInterest := 0.0
		totalInvestment := 0.0
		if deps.OrderService != nil {
			orders, orderErr := deps.OrderService.GetOrders(r.Context(), userID)
			if orderErr != nil {
				Error(w, http.StatusInternalServerError, orderErr.Error())
				return
			}
			activitiesCount = len(orders)
			for _, order := range orders {
				switch order.Type {
				case model.OrderTypeBuy:
					totalInvestment += order.Quantity * order.UnitPrice
				case model.OrderTypeSell:
					totalInvestment -= order.Quantity * order.UnitPrice
				case model.OrderTypeDividend:
					totalDividend += order.Quantity * order.UnitPrice
				case model.OrderTypeInterest:
					totalInterest += order.Quantity * order.UnitPrice
				}
			}
		}
		totalValue := totalBalance + totalInvestment + totalDividend + totalInterest

		// Legacy parity: include aggregate keys expected by client/API contract.
		JSON(w, http.StatusOK, map[string]any{
			"accounts":                    accounts,
			"activitiesCount":             activitiesCount,
			"totalBalanceInBaseCurrency":  totalBalance,
			"totalDividendInBaseCurrency": totalDividend,
			"totalInterestInBaseCurrency": totalInterest,
			"totalValueInBaseCurrency":    totalValue,
		})
	}
}

// CreateAccount handles POST /api/v1/account. If the request body supplies an
// id, that value is used verbatim; otherwise a new UUID is generated. This
// client-supplied-ID behaviour matches the NestJS implementation and is required
// for import/restore flows where the client must control IDs.
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

// DeleteAccount handles DELETE /api/v1/account/{id}. Ownership enforcement is
// delegated to AccountService.DeleteAccount which scopes the delete by both
// account ID and user ID, so a user cannot delete another user's account.
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

		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
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
		if deps == nil || deps.AccountBalanceService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID, err := resolveUserID(r, deps)
		if err != nil {
			writeResolveUserError(w, err)
			return
		}

		accountID := r.PathValue("id")
		balances, err := deps.AccountBalanceService.GetAccountBalances(r.Context(), accountID, userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if balances == nil {
			balances = []model.AccountBalance{}
		}
		JSON(w, http.StatusOK, map[string]any{"balances": balances})
	}
}

// TransferBalance handles POST /api/v1/account/transfer-balance. Before
// delegating to the service it validates that:
//   - both accounts belong to the authenticated user,
//   - source and destination are distinct,
//   - the source account has sufficient balance.
//
// WHY: These checks mirror the NestJS controller pre-conditions. Pushing them
// into the handler (not just the service) returns more specific HTTP error codes
// (400/404) rather than a generic 500 from a DB constraint violation.
//
// EDGE: The response is HTTP 200 with no body — this is legacy NestJS void
// return behaviour.
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
