// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// SubscriptionRepository handles persistence for Subscription records, which
// track paid access grants for Ghostfolio premium features. A subscription is
// valid while its expiresAt timestamp is in the future; expired rows are kept
// for billing history but are excluded from all active queries.
//
// NOTE: this repository is read-only from the application perspective — write
// operations are performed directly by the payment/admin layer, not via the
// API server.
type SubscriptionRepository struct {
	db db.DBTX
}

// NewSubscriptionRepository constructs a SubscriptionRepository backed by the
// provided database handle.
func NewSubscriptionRepository(db db.DBTX) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// subscriptionColumns is the canonical SELECT list for Subscription rows.
const subscriptionColumns = `id, "createdAt", "expiresAt", price, "updatedAt", "userId"`

// FindActiveByUserID returns the most-recently-expiring active subscription
// for a user. Returns ErrNoRows when the user has no valid subscription.
// WHY: LIMIT 1 with ORDER BY expiresAt DESC surfaces the longest-remaining
// subscription when a user holds overlapping grants (e.g. a gift code on top
// of a recurring subscription).
func (r *SubscriptionRepository) FindActiveByUserID(ctx context.Context, userID string) (model.Subscription, error) {
	var item model.Subscription
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+subscriptionColumns+`
		 FROM "Subscription"
		 WHERE "userId" = $1 AND "expiresAt" > NOW()
		 ORDER BY "expiresAt" DESC
		 LIMIT 1`, userID)
	if err != nil {
		return model.Subscription{}, fmt.Errorf("query active subscription: %w", err)
	}
	return item, nil
}

// FindAllActive returns every non-expired subscription across all users,
// ordered with the furthest-future expiry first. Used by admin dashboards to
// audit active subscribers.
func (r *SubscriptionRepository) FindAllActive(ctx context.Context) ([]model.Subscription, error) {
	var items []model.Subscription
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+subscriptionColumns+`
		 FROM "Subscription"
		 WHERE "expiresAt" > NOW()
		 ORDER BY "expiresAt" DESC`)
	if err != nil {
		return nil, fmt.Errorf("query active subscriptions: %w", err)
	}
	return items, nil
}
