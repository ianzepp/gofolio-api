package service

import (
	"context"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// OrderService manages trading activities (buy, sell, dividend, fee, etc.)
// that represent individual portfolio events. In Ghostfolio terminology an
// "Order" and an "Activity" are the same entity — the DB table is "Order" but
// the API and UI surface it as "Activity". Methods that accept a userID
// parameter enforce ownership at the DB level rather than in service code.
type OrderService struct {
	db *sqlx.DB
}

// NewOrderService constructs an OrderService backed by the given pool.
func NewOrderService(db *sqlx.DB) *OrderService {
	return &OrderService{db: db}
}

// GetOrders returns all activities belonging to the given user, unfiltered and
// unsorted (ordering is left to the caller or a higher-level service).
func (s *OrderService) GetOrders(ctx context.Context, userID string) ([]model.Order, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.FindByUserID(ctx, userID)
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.FindByID(ctx, id)
}

// GetOrderByUserID fetches a single order scoped to the given user, returning
// an error if the order does not exist or belongs to a different user.
func (s *OrderService) GetOrderByUserID(ctx context.Context, id string, userID string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.FindByIDAndUserID(ctx, id, userID)
}

func (s *OrderService) CreateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.Create(ctx, order)
}

func (s *OrderService) UpdateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.Update(ctx, order)
}

func (s *OrderService) UpdateOrderByUserID(ctx context.Context, order model.Order, userID string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.UpdateByUserID(ctx, order, userID)
}

func (s *OrderService) DeleteOrder(ctx context.Context, id string) error {
	repo := repository.NewOrderRepository(s.db)
	return repo.Delete(ctx, id)
}

func (s *OrderService) DeleteOrderByUserID(ctx context.Context, id string, userID string) (int64, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.DeleteByIDAndUserID(ctx, id, userID)
}

// DeleteOrdersByUserID deletes all activities for a user, returning the number
// of rows removed. This is used during account deletion and bulk data clearing.
func (s *OrderService) DeleteOrdersByUserID(ctx context.Context, userID string) (int64, error) {
	repo := repository.NewOrderRepository(s.db)
	return repo.DeleteByUserID(ctx, userID)
}

// UpdateTagsForHolding replaces all tag associations for orders matching a
// user's holding (by dataSource+symbol).
func (s *OrderService) UpdateTagsForHolding(ctx context.Context, userID, dataSource, symbol string, tagIDs []string) error {
	repo := repository.NewOrderRepository(s.db)

	orderIDs, err := repo.FindIDsByUserAndSymbol(ctx, userID, dataSource, symbol)
	if err != nil {
		return err
	}

	return repo.ReplaceTagsForOrders(ctx, orderIDs, tagIDs)
}
