package repository

import (
	"context"
	"fmt"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/jackc/pgx/v5"
)

type AnalyticsRepository struct {
	db db.DBTX
}

func NewAnalyticsRepository(db db.DBTX) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

const analyticsColumns = `"userId", "activityCount", country,
	"dataProviderGhostfolioDailyRequests", "lastRequestAt", "updatedAt"`

func (r *AnalyticsRepository) FindByUserID(ctx context.Context, userID string) (model.Analytics, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+analyticsColumns+`
		 FROM "Analytics"
		 WHERE "userId" = $1`, userID)
	if err != nil {
		return model.Analytics{}, fmt.Errorf("query analytics: %w", err)
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[model.Analytics])
}

func (r *AnalyticsRepository) FindAll(ctx context.Context) ([]model.Analytics, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+analyticsColumns+`
		 FROM "Analytics"
		 ORDER BY "lastRequestAt" DESC`)
	if err != nil {
		return nil, fmt.Errorf("query all analytics: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Analytics])
}
