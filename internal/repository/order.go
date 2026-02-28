package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type OrderRepository struct {
	db db.DBTX
}

func NewOrderRepository(db db.DBTX) *OrderRepository {
	return &OrderRepository{db: db}
}

const orderColumns = `id, "accountId", "accountUserId", comment, "createdAt",
	currency, date, fee, "isDraft", quantity, "symbolProfileId",
	type, "unitPrice", "updatedAt", "userId"`

func (r *OrderRepository) FindByUserID(ctx context.Context, userID string) ([]model.Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+orderColumns+`
		 FROM "Order"
		 WHERE "userId" = $1
		 ORDER BY date DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query orders: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Order])
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (model.Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+orderColumns+`
		 FROM "Order"
		 WHERE id = $1`, id)
	if err != nil {
		return model.Order{}, fmt.Errorf("query order: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Order])
}

func (r *OrderRepository) Create(ctx context.Context, order model.Order) (model.Order, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "Order" (id, "accountId", "accountUserId", comment,
		                      currency, date, fee, "isDraft", quantity,
		                      "symbolProfileId", type, "unitPrice", "userId")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING `+orderColumns,
		order.ID, order.AccountId, order.AccountUserId, order.Comment,
		order.Currency, order.Date, order.Fee, order.IsDraft, order.Quantity,
		order.SymbolProfileId, order.Type, order.UnitPrice, order.UserId)
	if err != nil {
		return model.Order{}, fmt.Errorf("insert order: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Order])
}

func (r *OrderRepository) Update(ctx context.Context, order model.Order) (model.Order, error) {
	rows, err := r.db.Query(ctx,
		`UPDATE "Order"
		 SET "accountId" = $2, "accountUserId" = $3, comment = $4,
		     currency = $5, date = $6, fee = $7, "isDraft" = $8,
		     quantity = $9, "symbolProfileId" = $10, type = $11,
		     "unitPrice" = $12
		 WHERE id = $1
		 RETURNING `+orderColumns,
		order.ID, order.AccountId, order.AccountUserId, order.Comment,
		order.Currency, order.Date, order.Fee, order.IsDraft, order.Quantity,
		order.SymbolProfileId, order.Type, order.UnitPrice)
	if err != nil {
		return model.Order{}, fmt.Errorf("update order: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Order])
}

func (r *OrderRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM "Order" WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete order: %w", err)
	}
	return nil
}

func (r *OrderRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM "Order" WHERE "userId" = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("delete orders by user: %w", err)
	}
	return tag.RowsAffected(), nil
}
