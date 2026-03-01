package service

import (
	"context"
	"encoding/json"

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

// OrderFilters holds the optional filter parameters for bulk order operations.
// Mirrors the legacy NestJS deleteOrders flow: fetch matching orders, then
// delete by collected IDs.
type OrderFilters struct {
	Accounts     []string
	AssetClasses []string
	Tags         []string
}

// DeleteOrdersWithFilters deletes orders matching the given filters for a user.
// If no filters are provided, all orders are deleted (matching legacy behavior).
// Returns the count of deleted rows.
func (s *OrderService) DeleteOrdersWithFilters(ctx context.Context, userID string, filters OrderFilters) (int64, error) {
	hasFilters := len(filters.Accounts) > 0 || len(filters.AssetClasses) > 0 || len(filters.Tags) > 0

	if !hasFilters {
		return s.DeleteOrdersByUserID(ctx, userID)
	}

	// Fetch orders with profile data to apply filters in Go (matching legacy
	// pattern: query with filters → collect IDs → deleteMany by IDs).
	repo := repository.NewOrderRepository(s.db)
	orders, err := repo.FindActiveByUserIDWithProfile(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Also include drafts — legacy's deleteOrders passes includeDrafts: true.
	allOrders, err := repo.FindByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Build a profile lookup from the joined query for filter matching.
	type profileInfo struct {
		AccountID  *string
		AssetClass *string
		TagIDs     []string
	}
	profileByID := make(map[string]profileInfo, len(orders))
	for _, o := range orders {
		var tagIDs []string
		if len(o.TagIDs) > 0 {
			_ = json.Unmarshal(o.TagIDs, &tagIDs)
		}
		profileByID[o.ID] = profileInfo{
			AccountID:  o.AccountId,
			AssetClass: o.SPAssetClass,
			TagIDs:     tagIDs,
		}
	}

	// Collect IDs of orders matching filters.
	accountSet := toSet(filters.Accounts)
	assetClassSet := toSet(filters.AssetClasses)
	tagSet := toSet(filters.Tags)

	var matchIDs []string
	for _, o := range allOrders {
		info, ok := profileByID[o.ID]
		if !ok {
			// Draft orders without profile join — include if no profile-dependent
			// filters are active.
			info = profileInfo{AccountID: o.AccountId}
		}

		if len(accountSet) > 0 {
			if info.AccountID == nil || !accountSet[*info.AccountID] {
				continue
			}
		}
		if len(assetClassSet) > 0 {
			if info.AssetClass == nil || !assetClassSet[*info.AssetClass] {
				continue
			}
		}
		if len(tagSet) > 0 {
			if !anyInSet(info.TagIDs, tagSet) {
				continue
			}
		}
		matchIDs = append(matchIDs, o.ID)
	}

	if len(matchIDs) == 0 {
		return 0, nil
	}

	return repo.DeleteByIDs(ctx, matchIDs)
}

func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}

func anyInSet(items []string, set map[string]bool) bool {
	for _, item := range items {
		if set[item] {
			return true
		}
	}
	return false
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
