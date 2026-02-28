package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExportService struct {
	pool *pgxpool.Pool
}

func NewExportService(pool *pgxpool.Pool) *ExportService {
	return &ExportService{pool: pool}
}

type ExportAccount struct {
	Balance    float64 `json:"balance"`
	Comment    *string `json:"comment"`
	Currency   *string `json:"currency"`
	ID         string  `json:"id"`
	IsExcluded bool    `json:"isExcluded"`
	Name       *string `json:"name"`
	PlatformId *string `json:"platformId"`
}

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

type ExportResponse struct {
	Meta       map[string]any   `json:"meta"`
	Accounts   []ExportAccount  `json:"accounts"`
	Activities []ExportActivity `json:"activities"`
}

func (s *ExportService) GetExport(ctx context.Context, userID string) (ExportResponse, error) {
	accountRepo := repository.NewAccountRepository(s.pool)
	orderRepo := repository.NewOrderRepository(s.pool)
	spRepo := repository.NewSymbolProfileRepository(s.pool)

	accounts, err := accountRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ExportResponse{}, fmt.Errorf("find accounts: %w", err)
	}

	orders, err := orderRepo.FindByUserID(ctx, userID)
	if err != nil {
		return ExportResponse{}, fmt.Errorf("find orders: %w", err)
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

	exportAccounts := make([]ExportAccount, 0, len(accounts))
	for _, a := range accounts {
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
