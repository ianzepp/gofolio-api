package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/order/order.controller.ts

Routes + request/response shape:
- DELETE /api/v1/order (DeleteOrders)
  Auth/permission: jwt / permissions.deleteOrder
  Response: number (count of deleted)
- GET /api/v1/order (GetOrders)
  Auth/permission: jwt / -
  Response: { activities: Order[], count: number }
- POST /api/v1/order (CreateOrder)
  Auth/permission: jwt / permissions.createOrder
  Response: Order
- DELETE /api/v1/order/{id} (DeleteOrder)
  Auth/permission: jwt / permissions.deleteOrder
  Response: Order
- GET /api/v1/order/{id} (GetOrder)
  Auth/permission: jwt / -
  Response: Order
- PUT /api/v1/order/{id} (UpdateOrder)
  Auth/permission: jwt / permissions.updateOrder
  Response: Order
*/

import (
	"errors"
	"net/http"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetOrders(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil {
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

		orders, err := deps.OrderService.GetOrders(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if orders == nil {
			orders = []model.Order{}
		}

		JSON(w, http.StatusOK, map[string]any{
			"activities": orders,
			"count":      len(orders),
		})
	}
}

func CreateOrder(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			AccountId       *string        `json:"accountId"`
			Comment         *string        `json:"comment"`
			Currency        *string        `json:"currency"`
			Date            time.Time      `json:"date"`
			Fee             float64        `json:"fee"`
			IsDraft         bool           `json:"isDraft"`
			Quantity        float64        `json:"quantity"`
			SymbolProfileId string         `json:"symbolProfileId"`
			Type            model.OrderType `json:"type"`
			UnitPrice       float64        `json:"unitPrice"`
		}
		if !Bind(w, r, &body) {
			return
		}

		var accountUserId *string
		if body.AccountId != nil {
			accountUserId = &userID
		}

		order := model.Order{
			ID:              uuid.New().String(),
			AccountId:       body.AccountId,
			AccountUserId:   accountUserId,
			Comment:         body.Comment,
			Currency:        body.Currency,
			Date:            body.Date,
			Fee:             body.Fee,
			IsDraft:         body.IsDraft,
			Quantity:        body.Quantity,
			SymbolProfileId: body.SymbolProfileId,
			Type:            body.Type,
			UnitPrice:       body.UnitPrice,
			UserId:          userID,
		}

		created, err := deps.OrderService.CreateOrder(r.Context(), order)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, created)
	}
}

func GetOrder(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil {
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
		order, err := deps.OrderService.GetOrderByUserID(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Legacy parity: returns not found when order is not visible to user.
				Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, order)
	}
}

func UpdateOrder(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")

		// Legacy parity: ownership/visibility check happens before update.
		_, err := deps.OrderService.GetOrderByUserID(r.Context(), id, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		var body struct {
			AccountId       *string        `json:"accountId"`
			AccountUserId   *string        `json:"accountUserId"`
			Comment         *string        `json:"comment"`
			Currency        *string        `json:"currency"`
			Date            time.Time      `json:"date"`
			Fee             float64        `json:"fee"`
			IsDraft         bool           `json:"isDraft"`
			Quantity        float64        `json:"quantity"`
			SymbolProfileId string         `json:"symbolProfileId"`
			Type            model.OrderType `json:"type"`
			UnitPrice       float64        `json:"unitPrice"`
		}
		if !Bind(w, r, &body) {
			return
		}

		order := model.Order{
			ID:              id,
			AccountId:       body.AccountId,
			AccountUserId:   body.AccountUserId,
			Comment:         body.Comment,
			Currency:        body.Currency,
			Date:            body.Date,
			Fee:             body.Fee,
			IsDraft:         body.IsDraft,
			Quantity:        body.Quantity,
			SymbolProfileId: body.SymbolProfileId,
			Type:            body.Type,
			UnitPrice:       body.UnitPrice,
		}

		updated, err := deps.OrderService.UpdateOrderByUserID(r.Context(), order, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, updated)
	}
}

func DeleteOrder(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		count, err := deps.OrderService.DeleteOrderByUserID(r.Context(), id, userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if count == 0 {
			// Legacy parity: returns forbidden when order is not owned by caller.
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

func DeleteOrders(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.OrderService == nil {
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

		// Legacy parity: bulk delete accepts filter params but current impl
		// deletes all orders for the user.
		_ = ParseFilters(r)
		count, err := deps.OrderService.DeleteOrdersByUserID(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Legacy parity: returns number of deleted rows.
		JSON(w, http.StatusOK, count)
	}
}
