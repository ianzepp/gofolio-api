package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminService struct {
	pool *pgxpool.Pool
}

func NewAdminService(pool *pgxpool.Pool) *AdminService {
	return &AdminService{pool: pool}
}

type AdminDashboard struct {
	ActivitiesCount int                    `json:"activitiesCount"`
	DataProviders   []AdminDataProvider    `json:"dataProviders"`
	Settings        map[string]interface{} `json:"settings"`
	UserCount       int                    `json:"userCount"`
	Version         string                 `json:"version"`
}

type AdminDataProvider struct {
	AssetProfileCount int    `json:"assetProfileCount"`
	DataSource        string `json:"dataSource,omitempty"`
	Name              string `json:"name,omitempty"`
}

func (s *AdminService) GetDashboard(ctx context.Context) (AdminDashboard, error) {
	orderRepo := repository.NewOrderRepository(s.pool)
	userRepo := repository.NewUserRepository(s.pool)
	propRepo := repository.NewPropertyRepository(s.pool)
	spRepo := repository.NewSymbolProfileRepository(s.pool)

	activitiesCount, err := orderRepo.CountAll(ctx)
	if err != nil {
		return AdminDashboard{}, fmt.Errorf("count activities: %w", err)
	}

	userCount, err := userRepo.CountAll(ctx)
	if err != nil {
		return AdminDashboard{}, fmt.Errorf("count users: %w", err)
	}

	// Load all properties as settings map
	props, err := propRepo.FindAll(ctx)
	settings := make(map[string]interface{})
	if err == nil {
		for _, p := range props {
			var parsed interface{}
			if json.Unmarshal([]byte(p.Value), &parsed) == nil {
				settings[p.Key] = parsed
			} else {
				settings[p.Key] = p.Value
			}
		}
	}

	// Count symbol profiles per data source
	dataSources := []string{
		"ALPHA_VANTAGE", "COINGECKO", "EOD_HISTORICAL_DATA",
		"FINANCIAL_MODELING_PREP", "GHOSTFOLIO", "GOOGLE_SHEETS",
		"MANUAL", "RAPID_API", "YAHOO",
	}
	dataProviders := make([]AdminDataProvider, 0, len(dataSources))
	for _, ds := range dataSources {
		count, _ := spRepo.CountByDataSource(ctx, ds)
		if count > 0 {
			dataProviders = append(dataProviders, AdminDataProvider{
				AssetProfileCount: count,
				DataSource:        ds,
				Name:              ds,
			})
		}
	}

	version := os.Getenv("GHOSTFOLIO_VERSION")

	return AdminDashboard{
		ActivitiesCount: activitiesCount,
		DataProviders:   dataProviders,
		Settings:        settings,
		UserCount:       userCount,
		Version:         version,
	}, nil
}

type AdminUser struct {
	AccountCount     int        `json:"accountCount"`
	ActivityCount    int        `json:"activityCount"`
	Country          *string    `json:"country"`
	CreatedAt        time.Time  `json:"createdAt"`
	DailyApiRequests int        `json:"dailyApiRequests"`
	Engagement       float64    `json:"engagement"`
	ID               string     `json:"id"`
	LastActivity     *time.Time `json:"lastActivity"`
	Provider         string     `json:"provider"`
	Role             string     `json:"role"`
}

type AdminUsersResponse struct {
	Count int         `json:"count"`
	Users []AdminUser `json:"users"`
}

func (s *AdminService) GetUsers(ctx context.Context, skip int, take int) (AdminUsersResponse, error) {
	// Query users joined with analytics and counts
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u."createdAt", u.provider, u.role,
		        a."activityCount", a.country,
		        a."dataProviderGhostfolioDailyRequests",
		        a."lastRequestAt",
		        (SELECT COUNT(*) FROM "Account" WHERE "userId" = u.id) as "accountCount"
		 FROM "User" u
		 LEFT JOIN "Analytics" a ON a."userId" = u.id
		 ORDER BY COALESCE(a."lastRequestAt", u."createdAt") DESC
		 LIMIT $1 OFFSET $2`, take, skip)
	if err != nil {
		return AdminUsersResponse{}, fmt.Errorf("query admin users: %w", err)
	}
	defer rows.Close()

	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		var activityCount *int
		var country *string
		var dailyReq *int
		var lastReq *time.Time
		var accountCount int

		if err := rows.Scan(&u.ID, &u.CreatedAt, &u.Provider, &u.Role,
			&activityCount, &country, &dailyReq, &lastReq, &accountCount); err != nil {
			return AdminUsersResponse{}, fmt.Errorf("scan admin user: %w", err)
		}

		u.AccountCount = accountCount
		if activityCount != nil {
			u.ActivityCount = *activityCount
		}
		u.Country = country
		if dailyReq != nil {
			u.DailyApiRequests = *dailyReq
		}
		u.LastActivity = lastReq

		// engagement = activityCount / days since registration (min 1)
		daysSinceRegistration := time.Since(u.CreatedAt).Hours() / 24
		if daysSinceRegistration < 1 {
			daysSinceRegistration = 1
		}
		u.Engagement = float64(u.ActivityCount) / daysSinceRegistration

		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return AdminUsersResponse{}, fmt.Errorf("iterate admin users: %w", err)
	}
	if users == nil {
		users = []AdminUser{}
	}

	// Total count
	var total int
	_ = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "User"`).Scan(&total)

	return AdminUsersResponse{Count: total, Users: users}, nil
}

func (s *AdminService) GetUser(ctx context.Context, id string) (AdminUser, error) {
	var u AdminUser
	var activityCount *int
	var country *string
	var dailyReq *int
	var lastReq *time.Time
	var accountCount int

	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u."createdAt", u.provider, u.role,
		        a."activityCount", a.country,
		        a."dataProviderGhostfolioDailyRequests",
		        a."lastRequestAt",
		        (SELECT COUNT(*) FROM "Account" WHERE "userId" = u.id) as "accountCount"
		 FROM "User" u
		 LEFT JOIN "Analytics" a ON a."userId" = u.id
		 WHERE u.id = $1`, id).Scan(
		&u.ID, &u.CreatedAt, &u.Provider, &u.Role,
		&activityCount, &country, &dailyReq, &lastReq, &accountCount)
	if err != nil {
		return AdminUser{}, fmt.Errorf("query admin user: %w", err)
	}

	u.AccountCount = accountCount
	if activityCount != nil {
		u.ActivityCount = *activityCount
	}
	u.Country = country
	if dailyReq != nil {
		u.DailyApiRequests = *dailyReq
	}
	u.LastActivity = lastReq

	daysSinceRegistration := time.Since(u.CreatedAt).Hours() / 24
	if daysSinceRegistration < 1 {
		daysSinceRegistration = 1
	}
	u.Engagement = float64(u.ActivityCount) / daysSinceRegistration

	return u, nil
}

type AdminMarketDataItem struct {
	ActivitiesCount int               `json:"activitiesCount"`
	AssetClass      *model.AssetClass `json:"assetClass"`
	AssetSubClass   *model.AssetSubClass `json:"assetSubClass"`
	Comment         *string           `json:"comment"`
	CountriesCount  int               `json:"countriesCount"`
	Currency        string            `json:"currency"`
	DataSource      string            `json:"dataSource"`
	Date            *time.Time        `json:"date"`
	ID              string            `json:"id"`
	IsActive        bool              `json:"isActive"`
	MarketDataItemCount int           `json:"marketDataItemCount"`
	Name            *string           `json:"name"`
	SectorsCount    int               `json:"sectorsCount"`
	Symbol          string            `json:"symbol"`
}

type AdminMarketDataResponse struct {
	Count      int                   `json:"count"`
	MarketData []AdminMarketDataItem `json:"marketData"`
}

func (s *AdminService) GetMarketData(ctx context.Context, skip int, take int, query string, sortColumn string, sortDirection string) (AdminMarketDataResponse, error) {
	// Build WHERE clause for search filtering
	whereClause := ""
	args := []any{}
	argIdx := 1

	if query != "" {
		whereClause = fmt.Sprintf(` WHERE (sp.symbol ILIKE $%d OR sp.name ILIKE $%d OR sp.isin ILIKE $%d)`, argIdx, argIdx, argIdx)
		args = append(args, "%"+query+"%")
		argIdx++
	}

	// Count total
	var total int
	countSQL := `SELECT COUNT(*) FROM "SymbolProfile" sp` + whereClause
	_ = s.pool.QueryRow(ctx, countSQL, args...).Scan(&total)

	// Determine sort
	orderBy := `sp.symbol ASC`
	if sortColumn != "" {
		dir := "ASC"
		if sortDirection == "desc" {
			dir = "DESC"
		}
		// Whitelist allowed sort columns
		switch sortColumn {
		case "symbol", "name", "dataSource", "assetClass", "assetSubClass":
			orderBy = fmt.Sprintf(`sp."%s" %s`, sortColumn, dir)
		}
	}

	// Main query with aggregates
	mainSQL := fmt.Sprintf(`SELECT sp.id, sp."assetClass", sp."assetSubClass", sp.comment,
	        sp.countries, sp.currency, sp."dataSource", sp."isActive",
	        sp.name, sp.sectors, sp.symbol,
	        (SELECT COUNT(*) FROM "Order" o
	         WHERE o."symbolProfileId" = sp.id) as "activitiesCount",
	        (SELECT COUNT(*) FROM "MarketData" md
	         WHERE md."dataSource" = sp."dataSource" AND md.symbol = sp.symbol) as "marketDataItemCount",
	        (SELECT md.date FROM "MarketData" md
	         WHERE md."dataSource" = sp."dataSource" AND md.symbol = sp.symbol
	         ORDER BY md.date DESC LIMIT 1) as "latestDate"
	 FROM "SymbolProfile" sp
	 %s
	 ORDER BY %s
	 LIMIT $%d OFFSET $%d`, whereClause, orderBy, argIdx, argIdx+1)

	args = append(args, take, skip)

	rows, err := s.pool.Query(ctx, mainSQL, args...)
	if err != nil {
		return AdminMarketDataResponse{}, fmt.Errorf("query admin market data: %w", err)
	}
	defer rows.Close()

	items := []AdminMarketDataItem{}
	for rows.Next() {
		var item AdminMarketDataItem
		var countries json.RawMessage
		var sectors json.RawMessage
		var latestDate *time.Time

		if err := rows.Scan(&item.ID, &item.AssetClass, &item.AssetSubClass, &item.Comment,
			&countries, &item.Currency, &item.DataSource, &item.IsActive,
			&item.Name, &sectors, &item.Symbol,
			&item.ActivitiesCount, &item.MarketDataItemCount, &latestDate); err != nil {
			return AdminMarketDataResponse{}, fmt.Errorf("scan admin market data: %w", err)
		}

		item.Date = latestDate

		// Count countries and sectors from JSON arrays
		if len(countries) > 0 {
			var arr []interface{}
			if json.Unmarshal(countries, &arr) == nil {
				item.CountriesCount = len(arr)
			}
		}
		if len(sectors) > 0 {
			var arr []interface{}
			if json.Unmarshal(sectors, &arr) == nil {
				item.SectorsCount = len(arr)
			}
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminMarketDataResponse{}, fmt.Errorf("iterate admin market data: %w", err)
	}

	return AdminMarketDataResponse{Count: total, MarketData: items}, nil
}
