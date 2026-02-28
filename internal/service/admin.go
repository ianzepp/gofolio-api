// Package service implements the business logic layer for the Ghostfolio Go API
// server. Each file in this package corresponds to a domain area (admin, accounts,
// orders, etc.) and sits between the HTTP handler layer and the repository layer.
//
// Services own orchestration logic: they coordinate across multiple repositories,
// apply domain rules (filtering, aggregation, engagement scoring), and return
// response shapes suitable for direct JSON serialization. They do not own SQL —
// that responsibility belongs to the repository layer.
//
// This Go port aims for JSON-level parity with the original NestJS Ghostfolio
// implementation so that existing clients and tests continue to work unchanged.
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

// AdminService provides administrative operations: dashboard metrics, user
// management, and market data inspection. It is intentionally separate from
// user-facing services so that admin-only endpoints can be guarded at the
// handler level without leaking privileged aggregates into general queries.
type AdminService struct {
	pool *pgxpool.Pool
}

// NewAdminService constructs an AdminService backed by the given connection pool.
func NewAdminService(pool *pgxpool.Pool) *AdminService {
	return &AdminService{pool: pool}
}

// AdminDashboard is the top-level response for GET /api/v1/admin. It aggregates
// system-wide counters and configuration so that an admin can assess the health
// and usage of the instance at a glance without running separate queries.
type AdminDashboard struct {
	ActivitiesCount int                    `json:"activitiesCount"`
	DataProviders   []AdminDataProvider    `json:"dataProviders"`
	Settings        map[string]interface{} `json:"settings"`
	UserCount       int                    `json:"userCount"`
	Version         string                 `json:"version"`
}

// AdminDataProvider summarises how many asset profiles are stored for a single
// data source. Data sources with zero profiles are omitted from the response to
// keep the payload small and avoid surfacing unused integrations.
type AdminDataProvider struct {
	AssetProfileCount int    `json:"assetProfileCount"`
	DataSource        string `json:"dataSource,omitempty"`
	Name              string `json:"name,omitempty"`
}

// GetDashboard assembles the admin dashboard by running several independent
// aggregate queries and merging the results. Errors on the Properties and
// data-source counts are treated as non-fatal — a missing or unparseable
// property degrades gracefully to an empty settings map rather than failing
// the entire request.
//
// PHASE 1: Count activities and users (required — errors are returned).
// PHASE 2: Load Properties as key-value settings; JSON values are parsed into
//
//	their native types, raw strings are kept as-is.
//
// PHASE 3: Count SymbolProfiles per known data source; sources with zero
//
//	profiles are suppressed to match NestJS dashboard parity.
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

// AdminUser is the admin-scoped view of a user, enriched with Analytics data
// that is not exposed through the standard user endpoints. The Engagement field
// is a computed rate (activities per day since registration) that lets admins
// quickly identify power users and inactive accounts without post-processing.
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

// AdminUsersResponse wraps the paginated user list with a total Count so that
// clients can implement pagination without a separate HEAD request.
type AdminUsersResponse struct {
	Count int         `json:"count"`
	Users []AdminUser `json:"users"`
}

// GetUsers returns a paginated, analytics-enriched list of all users ordered by
// most recent activity. The total Count reflects the unfiltered user population
// so that clients can compute page counts correctly.
//
// WHY: Admin user lists join the Analytics table to surface activity and geo
// data that the standard user query deliberately omits for privacy reasons.
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
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM "User"`).Scan(&total); err != nil {
		return AdminUsersResponse{}, fmt.Errorf("count admin users: %w", err)
	}

	return AdminUsersResponse{Count: total, Users: users}, nil
}

// GetUser returns the analytics-enriched admin view for a single user by ID.
// It mirrors the per-row logic of GetUsers so that the detail endpoint returns
// identical field shapes to the list endpoint.
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

// AdminMarketDataItem is the admin view of a SymbolProfile, augmented with
// derived aggregate counts (activities, market data points, countries, sectors)
// that let admins identify stale or incomplete profiles without joining
// manually. CountriesCount and SectorsCount are computed from the JSONB arrays
// stored on the profile rather than normalised foreign-key tables, which
// matches the original NestJS schema design.
type AdminMarketDataItem struct {
	ActivitiesCount     int                  `json:"activitiesCount"`
	AssetClass          *model.AssetClass    `json:"assetClass"`
	AssetSubClass       *model.AssetSubClass `json:"assetSubClass"`
	Comment             *string              `json:"comment"`
	CountriesCount      int                  `json:"countriesCount"`
	Currency            string               `json:"currency"`
	DataSource          string               `json:"dataSource"`
	Date                *time.Time           `json:"date"`
	ID                  string               `json:"id"`
	IsActive            bool                 `json:"isActive"`
	MarketDataItemCount int                  `json:"marketDataItemCount"`
	Name                *string              `json:"name"`
	SectorsCount        int                  `json:"sectorsCount"`
	Symbol              string               `json:"symbol"`
}

// AdminMarketDataResponse wraps the paginated market data list with a total
// Count for client-side pagination, mirroring the AdminUsersResponse pattern.
type AdminMarketDataResponse struct {
	Count      int                   `json:"count"`
	MarketData []AdminMarketDataItem `json:"marketData"`
}

// GetMarketData returns a paginated, filterable, sortable list of SymbolProfiles
// enriched with aggregate counts. It supports full-text search across symbol,
// name, and ISIN, and column-level sorting with a whitelist to prevent SQL
// injection through dynamic ORDER BY construction.
//
// EDGE: sortColumn values outside the whitelist are silently ignored and fall
// back to the default "symbol ASC" ordering rather than returning an error,
// preserving backward compatibility with clients that pass unknown column names.
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
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return AdminMarketDataResponse{}, fmt.Errorf("count admin market data: %w", err)
	}

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
