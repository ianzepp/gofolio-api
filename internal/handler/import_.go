package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
)

// importActivity is the inbound shape for a single activity in a portfolio
// import. It uses dataSource+symbol (the export format) rather than the
// internal symbolProfileId, so the handler must resolve the profile on the fly.
type importActivity struct {
	AccountId  *string         `json:"accountId"`
	Comment    *string         `json:"comment"`
	Currency   *string         `json:"currency"`
	DataSource string          `json:"dataSource"`
	Date       time.Time       `json:"date"`
	Fee        float64         `json:"fee"`
	Quantity   float64         `json:"quantity"`
	Symbol     string          `json:"symbol"`
	Type       model.OrderType `json:"type"`
	UnitPrice  float64         `json:"unitPrice"`
}

// importRequest is the top-level request body for POST /import.
type importRequest struct {
	Activities []importActivity `json:"activities"`
}

// HandleCreateImport imports portfolio activities from a Ghostfolio-format JSON
// payload. Each activity's dataSource+symbol pair is resolved to a SymbolProfile
// (auto-created if missing), then an Order is inserted for the authenticated user.
func HandleCreateImport(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil || deps.DB == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body importRequest
		if !Bind(w, r, &body) {
			return
		}

		if len(body.Activities) == 0 {
			JSON(w, http.StatusCreated, map[string]any{"activities": []any{}})
			return
		}

		spRepo := repository.NewSymbolProfileRepository(deps.DB)
		created := make([]model.Order, 0, len(body.Activities))

		for _, act := range body.Activities {
			if act.DataSource == "" || act.Symbol == "" {
				Error(w, http.StatusBadRequest, "each activity requires dataSource and symbol")
				return
			}

			// Resolve dataSource+symbol → SymbolProfile, creating if absent.
			sp, err := spRepo.FindByDataSourceAndSymbol(r.Context(), act.DataSource, act.Symbol)
			if errors.Is(err, sql.ErrNoRows) {
				sp, err = spRepo.Create(r.Context(), model.SymbolProfile{
					DataSource: model.DataSource(act.DataSource),
					Symbol:     act.Symbol,
					Currency:   currencyOrDefault(act.Currency),
				})
			}
			if err != nil {
				Error(w, http.StatusInternalServerError, "resolve symbol profile: "+err.Error())
				return
			}

			var accountUserId *string
			if act.AccountId != nil {
				accountUserId = &userID
			}

			order := model.Order{
				ID:              uuid.New().String(),
				AccountId:       act.AccountId,
				AccountUserId:   accountUserId,
				Comment:         act.Comment,
				Currency:        act.Currency,
				Date:            act.Date,
				Fee:             act.Fee,
				Quantity:        act.Quantity,
				SymbolProfileId: sp.ID,
				Type:            act.Type,
				UnitPrice:       act.UnitPrice,
				UserId:          userID,
			}

			inserted, err := deps.OrderService.CreateOrder(r.Context(), order)
			if err != nil {
				Error(w, http.StatusInternalServerError, "create order: "+err.Error())
				return
			}
			created = append(created, inserted)
		}

		JSON(w, http.StatusCreated, map[string]any{"activities": created})
	}
}

// HandleGetImportDividends returns existing dividend-type orders for the
// authenticated user that match the given dataSource+symbol, formatted as
// importable activities.
func HandleGetImportDividends(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil || deps.DB == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		spRepo := repository.NewSymbolProfileRepository(deps.DB)
		sp, err := spRepo.FindByDataSourceAndSymbol(r.Context(), dataSource, symbol)
		if errors.Is(err, sql.ErrNoRows) {
			JSON(w, http.StatusOK, map[string]any{"activities": []any{}})
			return
		}
		if err != nil {
			Error(w, http.StatusInternalServerError, "lookup symbol profile: "+err.Error())
			return
		}

		orders, err := deps.OrderService.GetOrders(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, "fetch orders: "+err.Error())
			return
		}

		dividends := make([]model.Order, 0)
		for _, o := range orders {
			if o.SymbolProfileId == sp.ID && o.Type == model.OrderTypeDividend {
				dividends = append(dividends, o)
			}
		}

		JSON(w, http.StatusOK, map[string]any{"activities": dividends})
	}
}

// currencyOrDefault returns the pointed-to currency string, or "USD" if nil.
func currencyOrDefault(c *string) string {
	if c != nil && *c != "" {
		return *c
	}
	return "USD"
}
