package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderService manages trading activities (buy, sell, dividend, fee, etc.)
// that represent individual portfolio events. In Ghostfolio terminology an
// "Order" and an "Activity" are the same entity — the DB table is "Order" but
// the API and UI surface it as "Activity". Methods that accept a userID
// parameter enforce ownership at the DB level rather than in service code.
type OrderService struct {
	pool *pgxpool.Pool
}

// NewOrderService constructs an OrderService backed by the given pool.
func NewOrderService(pool *pgxpool.Pool) *OrderService {
	return &OrderService{pool: pool}
}

// GetOrders returns all activities belonging to the given user, unfiltered and
// unsorted (ordering is left to the caller or a higher-level service).
func (s *OrderService) GetOrders(ctx context.Context, userID string) ([]model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.FindByUserID(ctx, userID)
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.FindByID(ctx, id)
}

// GetOrderByUserID fetches a single order scoped to the given user, returning
// an error if the order does not exist or belongs to a different user.
func (s *OrderService) GetOrderByUserID(ctx context.Context, id string, userID string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.FindByIDAndUserID(ctx, id, userID)
}

func (s *OrderService) CreateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.Create(ctx, order)
}

func (s *OrderService) UpdateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.Update(ctx, order)
}

func (s *OrderService) UpdateOrderByUserID(ctx context.Context, order model.Order, userID string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.UpdateByUserID(ctx, order, userID)
}

func (s *OrderService) DeleteOrder(ctx context.Context, id string) error {
	repo := repository.NewOrderRepository(s.pool)
	return repo.Delete(ctx, id)
}

func (s *OrderService) DeleteOrderByUserID(ctx context.Context, id string, userID string) (int64, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.DeleteByIDAndUserID(ctx, id, userID)
}

// DeleteOrdersByUserID deletes all activities for a user, returning the number
// of rows removed. This is used during account deletion and bulk data clearing.
func (s *OrderService) DeleteOrdersByUserID(ctx context.Context, userID string) (int64, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.DeleteByUserID(ctx, userID)
}
