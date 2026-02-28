package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type SubscriptionRepository struct {
	db db.DBTX
}

func NewSubscriptionRepository(db db.DBTX) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

const subscriptionColumns = `id, "createdAt", "expiresAt", price, "updatedAt", "userId"`

func (r *SubscriptionRepository) FindActiveByUserID(ctx context.Context, userID string) (model.Subscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+subscriptionColumns+`
		 FROM "Subscription"
		 WHERE "userId" = $1 AND "expiresAt" > NOW()
		 ORDER BY "expiresAt" DESC
		 LIMIT 1`, userID)
	if err != nil {
		return model.Subscription{}, fmt.Errorf("query active subscription: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Subscription])
}

func (r *SubscriptionRepository) FindAllActive(ctx context.Context) ([]model.Subscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+subscriptionColumns+`
		 FROM "Subscription"
		 WHERE "expiresAt" > NOW()
		 ORDER BY "expiresAt" DESC`)
	if err != nil {
		return nil, fmt.Errorf("query active subscriptions: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Subscription])
}
