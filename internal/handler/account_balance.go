// Package handler — account_balance.go implements handlers for
// POST /api/v1/account-balance and DELETE /api/v1/account-balance/{id}.
//
// Account balances are timestamped snapshots of a cash balance for a given
// account. The upsert semantic (CreateOrUpdate by accountId+date) is intentional:
// if the user submits a balance for the same account on the same day, it replaces
// the previous record rather than creating a duplicate. This mirrors the NestJS
// Prisma upsert and avoids duplicates when a user corrects a daily entry.
//
// Date handling: the client may send either YYYY-MM-DD or RFC3339 timestamps.
// All dates are normalised to midnight UTC (via repository.ResetHours) so that
// upsert keys are stable regardless of the time component in the submitted value.
//
// Ownership: both handlers verify that the account belongs to the authenticated
// user before operating on balance records, preventing cross-user data access.
package handler

import (
	"errors"
	"net/http"
	"time"

	"database/sql"
	"github.com/google/uuid"
	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
)

// HandleCreateAccountBalance handles POST /api/v1/account-balance. It validates
// account ownership before upserting the balance record. If a balance already
// exists for the same (accountId, date) pair it is overwritten, not duplicated.
//
// PHASE 1: Validate account ownership via AccountService.
// PHASE 2: Parse and normalise the date to midnight UTC.
// PHASE 3: Upsert the balance record.
func HandleCreateAccountBalance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountBalanceService == nil || deps.AccountService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			AccountId string  `json:"accountId"`
			Balance   float64 `json:"balance"`
			Date      string  `json:"date"`
		}
		if !Bind(w, r, &body) {
			return
		}

		if body.AccountId == "" || body.Date == "" {
			Error(w, http.StatusBadRequest, "accountId and date are required")
			return
		}

		// Validate account ownership (matches NestJS controller check).
		_, err := deps.AccountService.GetAccount(r.Context(), body.AccountId, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}

		date, parseErr := time.Parse("2006-01-02", body.Date)
		if parseErr != nil {
			// Try ISO8601 with time component
			date, parseErr = time.Parse(time.RFC3339, body.Date)
			if parseErr != nil {
				Error(w, http.StatusBadRequest, "invalid date format")
				return
			}
		}
		date = repository.ResetHours(date)

		ab := model.AccountBalance{
			ID:        uuid.New().String(),
			AccountId: body.AccountId,
			UserId:    userID,
			Date:      date,
			Value:     body.Balance,
		}

		created, err := deps.AccountBalanceService.CreateOrUpdateAccountBalance(r.Context(), ab)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, created)
	}
}

// HandleDeleteAccountBalance handles DELETE /api/v1/account-balance/{id}.
// A pre-flight ownership check returns 403 if the record does not exist or does
// not belong to the authenticated user, avoiding leakage of balance IDs.
func HandleDeleteAccountBalance(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AccountBalanceService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		if _, err := deps.AccountBalanceService.GetAccountBalance(r.Context(), id, userID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}

		deleted, err := deps.AccountBalanceService.DeleteAccountBalance(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
		JSON(w, http.StatusOK, deleted)
	}
}
