package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// ExportService assembles a portable data export for a user's portfolio. The
// export format is the canonical Ghostfolio interchange format: a versioned JSON
// envelope containing accounts and activities, suitable for backup, migration,
// and re-import. Filters allow clients to export subsets (e.g. a single
// account or a single symbol) without downloading the entire portfolio.
type ExportService struct {
	db *sqlx.DB
}

// NewExportService constructs an ExportService backed by the given pool.
func NewExportService(db *sqlx.DB) *ExportService {
	return &ExportService{db: db}
}

// ExportAccount is the portable representation of a brokerage/cash account.
// Internal fields like userId are intentionally omitted so that exported data
// can be re-imported into a different Ghostfolio instance without ID conflicts.
type ExportAccount struct {
	Balance    float64 `json:"balance"`
	Comment    *string `json:"comment"`
	Currency   *string `json:"currency"`
	ID         string  `json:"id"`
	IsExcluded bool    `json:"isExcluded"`
	Name       *string `json:"name"`
	PlatformId *string `json:"platformId"`
}

// ExportActivity is the portable representation of a trading or income event.
// It uses symbol+dataSource rather than the internal symbolProfileId, making it
// provider-addressable and human-readable in exported JSON files.
type ExportActivity struct {
	AccountId  *string   `json:"accountId"`
	Comment    *string   `json:"comment"`
	Currency   *string   `json:"currency"`
	DataSource string    `json:"dataSource"`
	Date       time.Time `json:"date"`
	Fee        float64   `json:"fee"`
	Quantity   float64   `json:"quantity"`
	Symbol     string    `json:"symbol"`
	Type       string    `json:"type"`
	UnitPrice  float64   `json:"unitPrice"`
}

// ExportResponse is the top-level export envelope. Meta carries a timestamp and
// schema version so that importers can detect format changes without inspecting
// the payload structure.
type ExportResponse struct {
	Meta       map[string]any   `json:"meta"`
	Accounts   []ExportAccount  `json:"accounts"`
	Activities []ExportActivity `json:"activities"`
}

// GetExport builds the filtered export payload for a user's portfolio.
// Filters are applied in two passes to preserve the invariant that exported
// accounts only contain accounts that have at least one matching activity.
//
// PHASE 1: Load all accounts and orders for the user from the DB.
// PHASE 2: Apply the account filter to narrow the order set.
// PHASE 3: Batch-fetch SymbolProfiles for the remaining orders to resolve
//
//	symbol and dataSource without per-order queries.
//
// PHASE 4: Apply activity, dataSource, and symbol filters.
// PHASE 5: Sort activities by date ascending (NestJS parity requirement).
// PHASE 6: Emit only accounts that have at least one activity in the result,
//
//	respecting the account filter if set.
//
// EDGE: An order whose SymbolProfile cannot be found is silently excluded
// rather than failing the entire export, preventing a stale DB state from
// blocking data retrieval.
func (s *ExportService) GetExport(
	ctx context.Context,
	userID string,
	filterAccountIDs []string,
	filterActivityIDs []string,
	filterDataSource string,
	filterSymbol string,
) (ExportResponse, error) {
	accountRepo := repository.NewAccountRepository(s.db)
	orderRepo := repository.NewOrderRepository(s.db)
	spRepo := repository.NewSymbolProfileRepository(s.db)

	accounts, err := accountRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ExportResponse{}, fmt.Errorf("find accounts: %w", err)
	}

	orders, err := orderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ExportResponse{}, fmt.Errorf("find orders: %w", err)
	}
	accountFilter := make(map[string]struct{}, len(filterAccountIDs))
	for _, id := range filterAccountIDs {
		accountFilter[id] = struct{}{}
	}
	activityFilter := make(map[string]struct{}, len(filterActivityIDs))
	for _, id := range filterActivityIDs {
		activityFilter[id] = struct{}{}
	}

	if len(accountFilter) > 0 {
		filteredOrders := make([]model.Order, 0, len(orders))
		for _, order := range orders {
			if order.AccountId == nil {
				continue
			}
			if _, ok := accountFilter[*order.AccountId]; ok {
				filteredOrders = append(filteredOrders, order)
			}
		}
		orders = filteredOrders
	}

	// Batch-fetch symbol profiles
	idSet := make(map[string]struct{})
	for _, o := range orders {
		idSet[o.SymbolProfileId] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	profiles, err := spRepo.FindByIDs(ctx, ids)
	if err != nil {
		return ExportResponse{}, fmt.Errorf("find symbol profiles: %w", err)
	}
	profileMap := make(map[string]model.SymbolProfile, len(profiles))
	for _, p := range profiles {
		profileMap[p.ID] = p
	}

	if len(activityFilter) > 0 || filterDataSource != "" || filterSymbol != "" {
		filteredOrders := make([]model.Order, 0, len(orders))
		for _, order := range orders {
			if len(activityFilter) > 0 {
				if _, ok := activityFilter[order.ID]; !ok {
					continue
				}
			}

			profile, ok := profileMap[order.SymbolProfileId]
			if !ok {
				continue
			}
			if filterDataSource != "" && string(profile.DataSource) != filterDataSource {
				continue
			}
			if filterSymbol != "" && profile.Symbol != filterSymbol {
				continue
			}

			filteredOrders = append(filteredOrders, order)
		}
		orders = filteredOrders
	}

	// Legacy parity: export activities are sorted by date ascending.
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].Date.Before(orders[j].Date)
	})

	accountsWithActivities := make(map[string]struct{}, len(orders))
	for _, order := range orders {
		if order.AccountId != nil {
			accountsWithActivities[*order.AccountId] = struct{}{}
		}
	}

	exportAccounts := make([]ExportAccount, 0, len(accounts))
	for _, a := range accounts {
		if len(accountFilter) > 0 {
			if _, ok := accountFilter[a.ID]; !ok {
				continue
			}
		}
		if len(activityFilter) > 0 {
			if _, ok := accountsWithActivities[a.ID]; !ok {
				continue
			}
		}
		exportAccounts = append(exportAccounts, ExportAccount{
			Balance:    a.Balance,
			Comment:    a.Comment,
			Currency:   a.Currency,
			ID:         a.ID,
			IsExcluded: a.IsExcluded,
			Name:       a.Name,
			PlatformId: a.PlatformId,
		})
	}

	exportActivities := make([]ExportActivity, 0, len(orders))
	for _, o := range orders {
		profile, ok := profileMap[o.SymbolProfileId]
		if !ok {
			continue
		}
		exportActivities = append(exportActivities, ExportActivity{
			AccountId:  o.AccountId,
			Comment:    o.Comment,
			Currency:   o.Currency,
			DataSource: string(profile.DataSource),
			Date:       o.Date,
			Fee:        o.Fee,
			Quantity:   o.Quantity,
			Symbol:     profile.Symbol,
			Type:       string(o.Type),
			UnitPrice:  o.UnitPrice,
		})
	}

	return ExportResponse{
		Meta:       map[string]any{"date": time.Now().UTC().Format(time.RFC3339), "version": "1"},
		Accounts:   exportAccounts,
		Activities: exportActivities,
	}, nil
}
