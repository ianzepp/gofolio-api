// Package repository — see order.go for the package-level doc comment.
package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jmoiron/sqlx"
)

// AnalyticsRepository handles persistence for the Analytics table, which
// stores per-user platform usage metrics. These metrics are used to enforce
// rate limits on the Ghostfolio data provider and to surface engagement
// statistics in the admin dashboard.
//
// The Analytics row is keyed by userId and is written by a background job
// rather than on every API request, so the data may lag real-time activity
// by up to a few minutes.
type AnalyticsRepository struct {
	db db.DBTX
}

// NewAnalyticsRepository constructs an AnalyticsRepository backed by the
// provided database handle.
func NewAnalyticsRepository(db db.DBTX) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// analyticsColumns is the canonical SELECT list for Analytics rows.
const analyticsColumns = `"userId", "activityCount", country,
	"dataProviderGhostfolioDailyRequests", "lastRequestAt", "updatedAt"`

// FindByUserID retrieves the analytics record for a single user. Returns
// ErrNoRows when the user has no analytics row yet (first-time or new user).
func (r *AnalyticsRepository) FindByUserID(ctx context.Context, userID string) (model.Analytics, error) {
	var item model.Analytics
	err := sqlx.GetContext(ctx, r.db,
		&item,
		`SELECT `+analyticsColumns+`
		 FROM "Analytics"
		 WHERE "userId" = $1`, userID)
	if err != nil {
		return model.Analytics{}, fmt.Errorf("query analytics: %w", err)
	}
	return item, nil
}

// FindAll returns analytics for every user ordered by most-recently-active
// first. Intended for admin dashboards only.
func (r *AnalyticsRepository) FindAll(ctx context.Context) ([]model.Analytics, error) {
	var items []model.Analytics
	err := sqlx.SelectContext(ctx, r.db,
		&items,
		`SELECT `+analyticsColumns+`
		 FROM "Analytics"
		 ORDER BY "lastRequestAt" DESC`)
	if err != nil {
		return nil, fmt.Errorf("query all analytics: %w", err)
	}
	return items, nil
}
