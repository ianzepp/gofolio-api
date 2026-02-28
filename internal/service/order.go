package service

import (
	"context"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderService struct {
	pool *pgxpool.Pool
}

func NewOrderService(pool *pgxpool.Pool) *OrderService {
	return &OrderService{pool: pool}
}

func (s *OrderService) GetOrders(ctx context.Context, userID string) ([]model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.FindByUserID(ctx, userID)
}

func (s *OrderService) GetOrder(ctx context.Context, id string) (model.Order, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.FindByID(ctx, id)
}

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

func (s *OrderService) DeleteOrdersByUserID(ctx context.Context, userID string) (int64, error) {
	repo := repository.NewOrderRepository(s.pool)
	return repo.DeleteByUserID(ctx, userID)
}
