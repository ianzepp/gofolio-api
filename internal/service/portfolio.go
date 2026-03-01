package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/ianzepp/gofolio-api/internal/calculator"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// PortfolioService orchestrates portfolio calculations.
type PortfolioService struct {
	db           *sqlx.DB
	exchangeRate *ExchangeRateService
	userService  *UserService
}

func assetKey(dataSource string, symbol string) string {
	return dataSource + "|" + symbol
}

// NewPortfolioService constructs a PortfolioService.
func NewPortfolioService(db *sqlx.DB, exchangeRate *ExchangeRateService, userService *UserService) *PortfolioService {
	return &PortfolioService{
		db:           db,
		exchangeRate: exchangeRate,
		userService:  userService,
	}
}

// buildCalculator creates a calculator from the user's orders and account balances.
// When filters is non-nil and non-empty, activities are filtered before being
// passed to the calculator so that the snapshot reflects only matching orders.
func (s *PortfolioService) buildCalculator(ctx context.Context, userID string, filters *PortfolioFilter) (*calculator.Calculator, string, error) {
	// Get user settings for base currency
	baseCurrency := "USD"
	settings, err := s.userService.GetSettings(ctx, userID)
	if err == nil && settings.Settings != nil {
		var settingsMap map[string]any
		if jsonErr := json.Unmarshal([]byte(*settings.Settings), &settingsMap); jsonErr == nil {
			if bc, ok := settingsMap["baseCurrency"].(string); ok && bc != "" {
				baseCurrency = bc
			}
		}
	}

	// Get orders with symbol profile
	orderRepo := repository.NewOrderRepository(s.db)
	ordersWithProfile, err := orderRepo.FindActiveByUserIDWithProfile(ctx, userID)
	if err != nil {
		return nil, baseCurrency, fmt.Errorf("fetch orders: %w", err)
	}

	// Convert to calculator activities
	activities := make([]calculator.Activity, 0, len(ordersWithProfile))
	for _, o := range ordersWithProfile {
		assetClass := ""
		if o.SPAssetClass != nil {
			assetClass = *o.SPAssetClass
		}
		assetSubClass := ""
		if o.SPAssetSubClass != nil {
			assetSubClass = *o.SPAssetSubClass
		}
		spUserID := ""
		if o.SPUserID != nil {
			spUserID = *o.SPUserID
		}
		accountID := ""
		if o.AccountId != nil {
			accountID = *o.AccountId
		}
		name := ""
		if o.SPName != nil {
			name = *o.SPName
		}
		url := ""
		if o.SPUrl != nil {
			url = *o.SPUrl
		}
		var countries []any
		if len(o.SPCountries) > 0 {
			_ = json.Unmarshal(o.SPCountries, &countries)
		}
		countries = enrichCountries(countries)
		var sectors []any
		if len(o.SPSectors) > 0 {
			_ = json.Unmarshal(o.SPSectors, &sectors)
		}
		var tags []calculator.Tag
		if len(o.TagIDs) > 0 {
			var tagIDs []string
			if json.Unmarshal(o.TagIDs, &tagIDs) == nil {
				tags = make([]calculator.Tag, 0, len(tagIDs))
				for _, tagID := range tagIDs {
					tags = append(tags, calculator.Tag{ID: tagID})
				}
			}
		}
		a := calculator.Activity{
			// Legacy parity: NestJS formats dates using local-time semantics.
			Date:              calculator.FormatDate(o.Date.In(time.Local)),
			Fee:               decimal.NewFromFloat(o.Fee),
			FeeInBaseCurrency: decimal.NewFromFloat(o.Fee), // Will be converted by calculator
			Quantity:          decimal.NewFromFloat(o.Quantity),
			UnitPrice:         decimal.NewFromFloat(o.UnitPrice),
			Type:              string(o.Type),
			Symbol:            o.SPSymbol,
			DataSource:        o.SPDataSource,
			Currency:          o.SPCurrency,
			AssetClass:        assetClass,
			AssetSubClass:     assetSubClass,
			UserID:            spUserID,
			Tags:              tags,
			AccountID:         accountID,
			Name:              name,
			URL:               url,
			Countries:         countries,
			Sectors:           sectors,
		}

		if !matchesFilter(a, filters) {
			continue
		}
		activities = append(activities, a)
	}

	// Get account balances
	abRepo := repository.NewAccountBalanceRepository(s.db)
	accountBalances, err := abRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, baseCurrency, fmt.Errorf("fetch account balances: %w", err)
	}

	balanceItems := make([]calculator.AccountBalanceItem, len(accountBalances))
	for i, ab := range accountBalances {
		balanceItems[i] = calculator.AccountBalanceItem{
			Date:  calculator.FormatDate(ab.Date),
			Value: decimal.NewFromFloat(ab.Value),
		}
	}

	calc := calculator.NewCalculator(activities, balanceItems, baseCurrency)
	return calc, baseCurrency, nil
}

// matchesFilter returns true if the activity passes all non-empty filter criteria.
func matchesFilter(a calculator.Activity, f *PortfolioFilter) bool {
	if f == nil || f.IsEmpty() {
		return true
	}
	if len(f.AccountIDs) > 0 {
		if a.AccountID == "" {
			return false
		}
		found := false
		for _, id := range f.AccountIDs {
			if a.AccountID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(f.AssetClasses) > 0 {
		found := false
		for _, ac := range f.AssetClasses {
			if a.AssetClass == ac {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if f.DataSource != "" && a.DataSource != f.DataSource {
		return false
	}
	if f.Symbol != "" && a.Symbol != f.Symbol {
		return false
	}
	if len(f.Tags) > 0 {
		if len(a.Tags) == 0 {
			return false
		}
		tagSet := map[string]bool{}
		for _, t := range a.Tags {
			tagSet[t.ID] = true
		}
		matched := false
		for _, tagID := range f.Tags {
			if tagSet[tagID] {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// computeSnapshot fetches market data and exchange rates, then runs the calculator.
func (s *PortfolioService) computeSnapshot(ctx context.Context, calc *calculator.Calculator, baseCurrency string) (*calculator.PortfolioSnapshot, error) {
	tps := calc.GetTransactionPoints()
	if len(tps) == 0 {
		return &calculator.PortfolioSnapshot{
			Errors:         []calculator.AssetIdentifier{},
			HistoricalData: []calculator.HistoricalDataItem{},
			Positions:      []calculator.TimelinePosition{},
		}, nil
	}

	lastTP := tps[len(tps)-1]

	// Collect unique currencies and (dataSource, symbol) pairs
	currencySet := map[string]bool{}
	type dsSymbol struct{ DataSource, Symbol string }
	var symbols []dsSymbol

	for _, item := range lastTP.Items {
		currencySet[item.Currency] = true
		if item.AssetSubClass != "CASH" {
			symbols = append(symbols, dsSymbol{item.DataSource, item.Symbol})
		}
	}

	currencies := make([]string, 0, len(currencySet))
	for c := range currencySet {
		currencies = append(currencies, c)
	}

	// Fetch exchange rates
	startDate := calc.GetStartDate()
	endDate := calc.GetEndDate()

	exchangeRates, err := s.exchangeRate.GetExchangeRatesByCurrency(ctx, currencies, baseCurrency, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("fetch exchange rates: %w", err)
	}

	// Fetch market data
	mdRepo := repository.NewMarketDataRepository(s.db)
	var mdItems []struct{ DataSource, Symbol string }
	for _, sym := range symbols {
		mdItems = append(mdItems, struct{ DataSource, Symbol string }{sym.DataSource, sym.Symbol})
	}

	marketDataRows, err := mdRepo.FindBySymbolsAndDateRange(ctx, mdItems, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("fetch market data: %w", err)
	}

	// Build MarketData map.
	// Legacy parity: key by symbol (without dataSource), matching NestJS ROAI
	// marketSymbolMap shape. This intentionally preserves legacy behavior.
	marketData := calculator.MarketData{}
	for _, md := range marketDataRows {
		dateStr := calculator.FormatDate(md.Date)
		if marketData[dateStr] == nil {
			marketData[dateStr] = map[string]decimal.Decimal{}
		}
		marketData[dateStr][md.Symbol] = decimal.NewFromFloat(md.MarketPrice)
	}

	snapshot := calc.ComputeSnapshot(marketData, exchangeRates)
	return snapshot, nil
}

// symbolProfileInfo holds enrichment data extracted from activities.
type symbolProfileInfo struct {
	Name          string
	AssetClass    string
	AssetSubClass string
	URL           string
	Countries     []any
	Sectors       []any
}

// GetHoldings returns portfolio holdings.
func (s *PortfolioService) GetHoldings(ctx context.Context, userID string, dateRange string, filters *PortfolioFilter) (*PortfolioHoldingsResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID, filters)
	if err != nil {
		return nil, err
	}

	// Build profile lookup from calculator activities
	profileMap := map[string]symbolProfileInfo{}
	for _, a := range calc.GetActivities() {
		key := assetKey(a.DataSource, a.Symbol)
		if _, ok := profileMap[key]; !ok {
			profileMap[key] = symbolProfileInfo{
				Name:          a.Name,
				AssetClass:    a.AssetClass,
				AssetSubClass: a.AssetSubClass,
				URL:           a.URL,
				Countries:     a.Countries,
				Sectors:       a.Sectors,
			}
		}
	}

	snapshot, err := s.computeSnapshot(ctx, calc, baseCurrency)
	if err != nil {
		return nil, err
	}

	// Calculate total value for allocation percentages
	totalValue := 0.0
	for _, pos := range snapshot.Positions {
		totalValue += pos.ValueInBaseCurrency.InexactFloat64()
	}

	holdings := make([]PortfolioHolding, 0, len(snapshot.Positions))
	for _, pos := range snapshot.Positions {
		// Legacy parity: default holdings/details exclude closed positions
		// (quantity == 0) unless an explicit closed-holding filter is applied.
		if pos.Quantity.Equal(decimal.Zero) {
			continue
		}

		allocation := 0.0
		if totalValue > 0 {
			allocation = pos.ValueInBaseCurrency.InexactFloat64() / totalValue
		}

		// Enrich with profile data
		displayName := pos.Symbol
		var assetClass, assetSubClass, url *string
		countries := []any{}
		sectors := []any{}
		if info, ok := profileMap[assetKey(pos.DataSource, pos.Symbol)]; ok {
			if info.Name != "" {
				displayName = info.Name
			}
			if info.AssetClass != "" {
				assetClass = &info.AssetClass
			}
			if info.AssetSubClass != "" {
				assetSubClass = &info.AssetSubClass
			}
			if info.URL != "" {
				url = &info.URL
			}
			if len(info.Countries) > 0 {
				countries = info.Countries
			}
			if len(info.Sectors) > 0 {
				sectors = info.Sectors
			}
		}

		h := PortfolioHolding{
			ActivitiesCount:        pos.ActivitiesCount,
			AllocationInPercentage: allocation,
			AssetClass:             assetClass,
			AssetSubClass:          assetSubClass,
			Currency:               pos.Currency,
			DataSource:             pos.DataSource,
			DateOfFirstActivity:    pos.DateOfFirstActivity,
			Dividend:               pos.Dividend.InexactFloat64(),
			Investment:             pos.Investment.InexactFloat64(),
			MarketPrice:            pos.MarketPrice,
			Name:                   displayName,
			Quantity:               pos.Quantity.InexactFloat64(),
			Symbol:                 pos.Symbol,
			Countries:              countries,
			Holdings:               []any{},
			Sectors:                sectors,
			Tags:                   tagsToAny(pos.Tags),
			URL:                    url,
		}

		vib := pos.ValueInBaseCurrency.InexactFloat64()
		h.ValueInBaseCurrency = &vib

		if pos.GrossPerformance != nil {
			v := pos.GrossPerformance.InexactFloat64()
			h.GrossPerformance = &v
		}
		if pos.GrossPerformancePercentage != nil {
			v := pos.GrossPerformancePercentage.InexactFloat64()
			h.GrossPerformancePercent = &v
		}
		if pos.GrossPerformancePercentageWithCurrencyEffect != nil {
			v := pos.GrossPerformancePercentageWithCurrencyEffect.InexactFloat64()
			h.GrossPerformancePercentWithCurrencyEffect = &v
		}
		if pos.GrossPerformanceWithCurrencyEffect != nil {
			v := pos.GrossPerformanceWithCurrencyEffect.InexactFloat64()
			h.GrossPerformanceWithCurrencyEffect = &v
		}
		if pos.NetPerformance != nil {
			v := pos.NetPerformance.InexactFloat64()
			h.NetPerformance = &v
		}
		if pos.NetPerformancePercentage != nil {
			v := pos.NetPerformancePercentage.InexactFloat64()
			h.NetPerformancePercent = &v
		}
		// Use the "max" value from netPerformancePercentageWithCurrencyEffectMap
		if m, ok := pos.NetPerformancePercentageWithCurrencyEffectMap["max"]; ok {
			v := m.InexactFloat64()
			h.NetPerformancePercentWithCurrencyEffect = &v
		}
		if m, ok := pos.NetPerformanceWithCurrencyEffectMap["max"]; ok {
			v := m.InexactFloat64()
			h.NetPerformanceWithCurrencyEffect = &v
		}

		holdings = append(holdings, h)
	}

	// Sort holdings by symbol
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].Symbol < holdings[j].Symbol
	})

	return &PortfolioHoldingsResponse{Holdings: holdings}, nil
}

// GetDetails returns portfolio details.
func (s *PortfolioService) GetDetails(ctx context.Context, userID string, dateRange string, filters *PortfolioFilter) (*PortfolioDetailsResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID, filters)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.computeSnapshot(ctx, calc, baseCurrency)
	if err != nil {
		return nil, err
	}

	holdingsResp, err := s.GetHoldings(ctx, userID, dateRange, filters)
	if err != nil {
		return nil, err
	}

	holdingsMap := make(map[string]PortfolioHolding)
	for _, h := range holdingsResp.Holdings {
		holdingsMap[h.Symbol] = h
	}

	return &PortfolioDetailsResponse{
		Accounts: map[string]AccountDetail{},
		HasError: snapshot.HasErrors,
		Holdings: holdingsMap,
	}, nil
}

// GetPerformance returns portfolio performance data.
func (s *PortfolioService) GetPerformance(ctx context.Context, userID string, dateRange string, filters *PortfolioFilter) (*PortfolioPerformanceResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID, filters)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.computeSnapshot(ctx, calc, baseCurrency)
	if err != nil {
		return nil, err
	}

	// Get date range
	start, end := calculator.GetIntervalFromDateRange(dateRange, calc.GetStartDate())
	chart := calc.GetPerformance(start, end, snapshot)

	// Build chart as []map[string]any for JSON
	chartMaps := make([]map[string]any, len(chart))
	for i, item := range chart {
		chartMaps[i] = map[string]any{
			"date":                              item.Date,
			"investmentValueWithCurrencyEffect": item.InvestmentValueWithCurrencyEffect,
			"netPerformance":                    item.NetPerformance,
			"netPerformanceInPercentage":        item.NetPerformanceInPercentage,
			"netPerformanceInPercentageWithCurrencyEffect": item.NetPerformanceInPercentageWithCurrencyEffect,
			"netPerformanceWithCurrencyEffect":             item.NetPerformanceWithCurrencyEffect,
			"netWorth":                                     item.NetWorth,
			"totalAccountBalance":                          item.TotalAccountBalance,
			"totalInvestment":                              item.TotalInvestment,
			"totalInvestmentValueWithCurrencyEffect":       item.TotalInvestmentValueWithCurrencyEffect,
			"value":                                        item.Value,
			"valueWithCurrencyEffect":                      item.ValueWithCurrencyEffect,
		}
	}

	// Extract performance from last chart entry
	perf := PerformanceSummary{
		CurrentValueInBaseCurrency: snapshot.CurrentValueInBaseCurrency.InexactFloat64(),
		TotalInvestment:            snapshot.TotalInvestment.InexactFloat64(),
	}

	if len(chart) > 0 {
		last := chart[len(chart)-1]
		perf.NetPerformance = last.NetPerformance
		perf.NetPerformancePercentage = last.NetPerformanceInPercentage
		perf.NetPerformancePercentageWithCurrencyEffect = last.NetPerformanceInPercentageWithCurrencyEffect
		perf.NetPerformanceWithCurrencyEffect = last.NetPerformanceWithCurrencyEffect
		perf.TotalInvestmentValueWithCurrencyEffect = last.TotalInvestmentValueWithCurrencyEffect
	}

	firstOrderDate := ""
	tps := calc.GetTransactionPoints()
	if len(tps) > 0 {
		firstOrderDate = tps[0].Date
	}

	return &PortfolioPerformanceResponse{
		Chart:          chartMaps,
		HasErrors:      snapshot.HasErrors,
		FirstOrderDate: firstOrderDate,
		Performance:    perf,
	}, nil
}

// GetDividends returns portfolio dividends grouped by date.
func (s *PortfolioService) GetDividends(ctx context.Context, userID string, dateRange string, groupBy string, filters *PortfolioFilter) (*PortfolioDividendsResponse, error) {
	calc, _, err := s.buildCalculator(ctx, userID, filters)
	if err != nil {
		return nil, err
	}

	// Date-range semantics are anchored to calculator start date for parity with
	// other portfolio endpoints.
	start, end := calculator.GetIntervalFromDateRange(dateRange, calc.GetStartDate())

	orderRepo := repository.NewOrderRepository(s.db)
	ordersWithProfile, err := orderRepo.FindActiveByUserIDWithProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch orders: %w", err)
	}

	type aggregate struct {
		date  string
		value float64
	}
	dividendsByDate := map[string]float64{}
	for _, order := range ordersWithProfile {
		if string(order.Type) != "DIVIDEND" {
			continue
		}
		if order.Date.Before(start) || order.Date.After(end) {
			continue
		}

		// Apply filters to dividend orders too
		if !filters.IsEmpty() {
			accountID := ""
			if order.AccountId != nil {
				accountID = *order.AccountId
			}
			assetClass := ""
			if order.SPAssetClass != nil {
				assetClass = *order.SPAssetClass
			}
			a := calculator.Activity{
				AccountID:  accountID,
				AssetClass: assetClass,
				DataSource: order.SPDataSource,
				Symbol:     order.SPSymbol,
			}
			if !matchesFilter(a, filters) {
				continue
			}
		}

		// Legacy parity: NestJS formats JS Date in local timezone. Mirror that by
		// formatting with time.Local instead of UTC-normalized calculator dates.
		date := order.Date.In(time.Local).Format("2006-01-02")
		if groupBy == "month" && len(date) >= 7 {
			date = date[:7] + "-01"
		} else if groupBy == "year" && len(date) >= 4 {
			date = date[:4] + "-01-01"
		}
		dividendsByDate[date] += order.Quantity * order.UnitPrice
	}

	items := make([]aggregate, 0, len(dividendsByDate))
	for date, value := range dividendsByDate {
		items = append(items, aggregate{date: date, value: value})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].date < items[j].date })

	dividends := make([]InvestmentItemResponse, 0, len(items))
	for _, item := range items {
		dividends = append(dividends, InvestmentItemResponse{
			Date:       item.date,
			Investment: item.value,
		})
	}

	return &PortfolioDividendsResponse{Dividends: dividends}, nil
}

// GetInvestments returns portfolio investment timeline.
func (s *PortfolioService) GetInvestments(ctx context.Context, userID string, dateRange string, groupBy string, filters *PortfolioFilter) (*PortfolioInvestmentsResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID, filters)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.computeSnapshot(ctx, calc, baseCurrency)
	if err != nil {
		return nil, err
	}

	start, end := calculator.GetIntervalFromDateRange(dateRange, calc.GetStartDate())

	// Filter historical data to range
	var filteredData []calculator.HistoricalDataItem
	for _, item := range snapshot.HistoricalData {
		d := calculator.ParseDate(item.Date)
		if !d.Before(start) && !d.After(end) {
			filteredData = append(filteredData, item)
		}
	}

	var investments []InvestmentItemResponse

	if groupBy != "" {
		grouped := calc.GetInvestmentsByGroup(filteredData, groupBy)
		investments = make([]InvestmentItemResponse, len(grouped))
		for i, g := range grouped {
			investments[i] = InvestmentItemResponse{
				Date:       g.Date,
				Investment: g.Investment,
			}
		}
	} else {
		// Return raw investment values
		for _, item := range filteredData {
			investments = append(investments, InvestmentItemResponse{
				Date:       item.Date,
				Investment: item.InvestmentValueWithCurrencyEffect,
			})
		}
	}

	if investments == nil {
		investments = []InvestmentItemResponse{}
	}

	return &PortfolioInvestmentsResponse{Investments: investments}, nil
}

// GetHolding returns a single holding detail.
// The response shape matches the legacy NestJS PortfolioHoldingResponse exactly:
// top-level scalar fields + a nested SymbolProfile object (capital S key).
func (s *PortfolioService) GetHolding(ctx context.Context, userID string, dataSource string, symbol string) (*PortfolioHoldingDetailResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.computeSnapshot(ctx, calc, baseCurrency)
	if err != nil {
		return nil, err
	}

	// Find the position
	var found *calculator.TimelinePosition
	for i := range snapshot.Positions {
		if snapshot.Positions[i].DataSource == dataSource && snapshot.Positions[i].Symbol == symbol {
			found = &snapshot.Positions[i]
			break
		}
	}

	if found == nil {
		return nil, nil // not found
	}

	// Load SymbolProfile from DB to build the nested object
	repo := repository.NewSymbolProfileRepository(s.db)
	profile, err := repo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		return nil, fmt.Errorf("load symbol profile for holding: %w", err)
	}

	// Build the nested SymbolProfile matching legacy EnhancedSymbolProfile
	var assetClass, assetSubClass *string
	if profile.AssetClass != nil {
		ac := string(*profile.AssetClass)
		assetClass = &ac
	}
	if profile.AssetSubClass != nil {
		asc := string(*profile.AssetSubClass)
		assetSubClass = &asc
	}

	var currency *string
	if profile.Currency != "" {
		currency = &profile.Currency
	}

	symProfile := &HoldingSymbolProfile{
		ActivitiesCount: found.ActivitiesCount,
		AssetClass:      assetClass,
		AssetSubClass:   assetSubClass,
		Comment:         profile.Comment,
		Countries:       ensureJSONArray(profile.Countries),
		CreatedAt:       profile.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		Currency:        currency,
		Cusip:           profile.Cusip,
		DataSource:      string(profile.DataSource),
		Figi:            profile.Figi,
		FigiComposite:   profile.FigiComposite,
		FigiShareClass:  profile.FigiShareClass,
		Holdings:        ensureJSONArray(profile.Holdings),
		ID:              profile.ID,
		IsActive:        profile.IsActive,
		Isin:            profile.Isin,
		Name:            profile.Name,
		Sectors:         ensureJSONArray(profile.Sectors),
		Symbol:          profile.Symbol,
		SymbolMapping:   ensureJSONObject(profile.SymbolMapping),
		UpdatedAt:       profile.UpdatedAt.Format("2006-01-02T15:04:05.000Z"),
		URL:             profile.Url,
	}

	resp := &PortfolioHoldingDetailResponse{
		ActivitiesCount:                            found.ActivitiesCount,
		AveragePrice:                               found.AveragePrice.InexactFloat64(),
		DateOfFirstActivity:                        found.DateOfFirstActivity,
		DividendInBaseCurrency:                     found.DividendInBaseCurrency.InexactFloat64(),
		DividendYieldPercent:                       0,
		DividendYieldPercentWithCurrencyEffect:     0,
		FeeInBaseCurrency:                          found.FeeInBaseCurrency.InexactFloat64(),
		HistoricalData:                             []map[string]any{},
		InvestmentInBaseCurrencyWithCurrencyEffect: found.InvestmentWithCurrencyEffect.InexactFloat64(),
		MarketPrice:                                found.MarketPrice,
		MarketPriceMax:                             found.MarketPrice,
		MarketPriceMin:                             found.MarketPrice,
		Quantity:                                   found.Quantity.InexactFloat64(),
		SymbolProfile:                              symProfile,
		Tags:                                       tagsToAny(found.Tags),
		Value:                                      found.ValueInBaseCurrency.InexactFloat64(),
		Performances: &HoldingPerformances{
			AllTimeHigh: HoldingAllTimeHigh{
				PerformancePercent: 0,
			},
		},
	}

	if found.GrossPerformance != nil {
		v := found.GrossPerformance.InexactFloat64()
		resp.GrossPerformance = &v
	}
	if found.GrossPerformancePercentage != nil {
		v := found.GrossPerformancePercentage.InexactFloat64()
		resp.GrossPerformancePercent = &v
	}
	if found.GrossPerformancePercentageWithCurrencyEffect != nil {
		v := found.GrossPerformancePercentageWithCurrencyEffect.InexactFloat64()
		resp.GrossPerformancePercentWithCurrencyEffect = &v
	}
	if found.GrossPerformanceWithCurrencyEffect != nil {
		v := found.GrossPerformanceWithCurrencyEffect.InexactFloat64()
		resp.GrossPerformanceWithCurrencyEffect = &v
	}
	if found.NetPerformance != nil {
		v := found.NetPerformance.InexactFloat64()
		resp.NetPerformance = &v
	}
	if found.NetPerformancePercentage != nil {
		v := found.NetPerformancePercentage.InexactFloat64()
		resp.NetPerformancePercent = &v
	}
	if m, ok := found.NetPerformancePercentageWithCurrencyEffectMap["max"]; ok {
		v := m.InexactFloat64()
		resp.NetPerformancePercentWithCurrencyEffect = &v
	}
	if m, ok := found.NetPerformanceWithCurrencyEffectMap["max"]; ok {
		v := m.InexactFloat64()
		resp.NetPerformanceWithCurrencyEffect = &v
	}

	return resp, nil
}

// GetReport returns portfolio report rules analysis (X-Ray).
// Mirrors the legacy NestJS getReport() with all 8 rule categories and 17 rules.
func (s *PortfolioService) GetReport(ctx context.Context, userID string) (map[string]any, error) {
	details, err := s.GetDetails(ctx, userID, "max", nil)
	if err != nil {
		return nil, fmt.Errorf("get details for report: %w", err)
	}

	// Count activities
	activityCount := 0
	for _, h := range details.Holdings {
		activityCount += h.ActivitiesCount
	}

	// Summary values
	cash := float64(0)
	fees := float64(0)
	totalInvestment := float64(0)
	if details.Summary != nil {
		cash = details.Summary.Cash
		fees = details.Summary.Fees
		totalInvestment = details.Summary.TotalInvestment
	}

	// Group holdings by currency and asset class for rules
	type groupEntry struct {
		investment float64
		value      float64
	}
	currencyGroups := map[string]*groupEntry{}
	assetClassGroups := map[string]*groupEntry{}

	for _, h := range details.Holdings {
		val := float64(0)
		if h.ValueInBaseCurrency != nil {
			val = *h.ValueInBaseCurrency
		}
		// Currency grouping
		cg, ok := currencyGroups[h.Currency]
		if !ok {
			cg = &groupEntry{}
			currencyGroups[h.Currency] = cg
		}
		cg.investment += h.Investment
		cg.value += val
		// Asset class grouping
		ac := "UNKNOWN"
		if h.AssetClass != nil {
			ac = *h.AssetClass
		}
		ag, ok := assetClassGroups[ac]
		if !ok {
			ag = &groupEntry{}
			assetClassGroups[ac] = ag
		}
		ag.investment += h.Investment
		ag.value += val
	}

	// Compute totals
	totalCurrencyValue := float64(0)
	for _, g := range currencyGroups {
		totalCurrencyValue += g.value
	}
	totalAssetValue := float64(0)
	for _, g := range assetClassGroups {
		totalAssetValue += g.value
	}

	// Account totals
	totalAccountValue := float64(0)
	for _, a := range details.Accounts {
		totalAccountValue += a.ValueInBaseCurrency
	}

	// Get base currency
	baseCurrency := "USD"
	settings, err := s.userService.GetSettings(ctx, userID)
	if err == nil && settings.Settings != nil {
		var settingsMap map[string]any
		if jsonErr := json.Unmarshal([]byte(*settings.Settings), &settingsMap); jsonErr == nil {
			if bc, ok := settingsMap["baseCurrency"].(string); ok && bc != "" {
				baseCurrency = bc
			}
		}
	}

	// Helper to make a rule result
	makeRule := func(key string, name string, value bool, isActive bool, config any) map[string]any {
		r := map[string]any{
			"key":      key,
			"name":     name,
			"isActive": isActive,
		}
		if isActive {
			r["value"] = value
			if config != nil {
				r["configuration"] = config
			}
		}
		return r
	}

	// Threshold config helper
	thresholdCfg := func(max, min, step float64, unit string, hasMax, hasMin bool) map[string]any {
		cfg := map[string]any{
			"threshold": map[string]any{
				"max":  max,
				"min":  min,
				"step": step,
				"unit": unit,
			},
		}
		if hasMax {
			cfg["thresholdMax"] = true
		}
		if hasMin {
			cfg["thresholdMin"] = true
		}
		return cfg
	}

	// --- Rule evaluations ---

	// 1. BuyingPower
	buyingPowerValue := cash > 0
	buyingPowerRule := makeRule("BuyingPower", "Buying power", buyingPowerValue, true,
		thresholdCfg(200000, 0, 1000, "", false, true))

	// 2. EmergencyFundSetup
	emergencyFundRule := makeRule("EmergencyFundSetup", "Emergency fund", false, true, nil)

	// 3. CurrencyClusterRisk
	baseCurrencyValue := float64(0)
	if g, ok := currencyGroups[baseCurrency]; ok {
		baseCurrencyValue = g.value
	}
	maxCurrencyInvestment := float64(0)
	maxCurrencyKey := ""
	for k, g := range currencyGroups {
		if g.investment > maxCurrencyInvestment {
			maxCurrencyInvestment = g.investment
			maxCurrencyKey = k
		}
	}
	_ = baseCurrencyValue
	ccBaseCurrencyValue := maxCurrencyKey == baseCurrency
	ccBaseCurrencyRule := makeRule("CurrencyClusterRiskBaseCurrencyCurrentInvestment",
		"Investment: Base currency", ccBaseCurrencyValue, true, nil)

	maxCurrencyValue := float64(0)
	for _, g := range currencyGroups {
		if g.value > maxCurrencyValue {
			maxCurrencyValue = g.value
		}
	}
	ccCurrentRatio := float64(0)
	if totalCurrencyValue > 0 {
		ccCurrentRatio = maxCurrencyValue / totalCurrencyValue
	}
	ccCurrentValue := ccCurrentRatio <= 0.5
	ccCurrentRule := makeRule("CurrencyClusterRiskCurrentInvestment",
		"Investment: Current", ccCurrentValue, true,
		thresholdCfg(1, 0, 0.01, "%", true, false))

	// 4. AssetClassClusterRisk
	equityValue := float64(0)
	if g, ok := assetClassGroups["EQUITY"]; ok {
		equityValue = g.value
	}
	equityRatio := float64(0)
	if totalAssetValue > 0 {
		equityRatio = equityValue / totalAssetValue
	}
	equityOk := equityRatio >= 0.78 && equityRatio <= 0.82
	equityRule := makeRule("AssetClassClusterRiskEquity", "Equity", equityOk, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))

	fixedIncomeValue := float64(0)
	if g, ok := assetClassGroups["FIXED_INCOME"]; ok {
		fixedIncomeValue = g.value
	}
	fixedIncomeRatio := float64(0)
	if totalAssetValue > 0 {
		fixedIncomeRatio = fixedIncomeValue / totalAssetValue
	}
	fixedIncomeOk := fixedIncomeRatio >= 0.18 && fixedIncomeRatio <= 0.22
	fixedIncomeRule := makeRule("AssetClassClusterRiskFixedIncome", "Fixed income", fixedIncomeOk, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))

	// 5. AccountClusterRisk
	numAccounts := len(details.Accounts)
	maxAccountValue := float64(0)
	for _, a := range details.Accounts {
		if a.ValueInBaseCurrency > maxAccountValue {
			maxAccountValue = a.ValueInBaseCurrency
		}
	}
	accountRatio := float64(0)
	if totalAccountValue > 0 {
		accountRatio = maxAccountValue / totalAccountValue
	}
	accountCurrentOk := numAccounts > 0 && accountRatio <= 0.5
	accountCurrentRule := makeRule("AccountClusterRiskCurrentInvestment",
		"Investment: Current", accountCurrentOk, true,
		thresholdCfg(1, 0, 0.01, "%", true, false))

	accountSingleOk := numAccounts > 1
	accountSingleRule := makeRule("AccountClusterRiskSingleAccount",
		"Single account", accountSingleOk, true, nil)

	// 6-7. Economic and Regional market cluster risks (stub with neutral defaults)
	// These require detailed country-level data from holdings which may not be fully
	// populated. Use safe defaults that evaluate to true (pass).
	econDevRule := makeRule("EconomicMarketClusterRiskDevelopedMarkets",
		"Developed markets", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))
	econEmgRule := makeRule("EconomicMarketClusterRiskEmergingMarkets",
		"Emerging markets", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))
	regAPRule := makeRule("RegionalMarketClusterRiskAsiaPacific",
		"Asia-Pacific", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))
	regEmgRule := makeRule("RegionalMarketClusterRiskEmergingMarkets",
		"Emerging markets", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))
	regEuRule := makeRule("RegionalMarketClusterRiskEurope",
		"Europe", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))
	regJpRule := makeRule("RegionalMarketClusterRiskJapan",
		"Japan", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))
	regNARule := makeRule("RegionalMarketClusterRiskNorthAmerica",
		"North America", true, true,
		thresholdCfg(1, 0, 0.01, "%", true, true))

	// 8. Fee ratios
	feeRatioInitial := float64(0)
	if totalInvestment > 0 {
		feeRatioInitial = fees / totalInvestment
	}
	feeInitialOk := feeRatioInitial <= 0.01
	feeInitialRule := makeRule("FeeRatioInitialInvestment",
		"Initial investment", feeInitialOk, true,
		thresholdCfg(0.1, 0, 0.0025, "%", true, false))

	feeVolumeRule := makeRule("FeeRatioTotalInvestmentVolume",
		"Total investment volume", feeInitialOk, true,
		thresholdCfg(0.1, 0, 0.0025, "%", true, false))

	// Build categories
	categories := []map[string]any{
		{"key": "liquidity", "name": "Liquidity", "rules": []any{buyingPowerRule}},
		{"key": "emergencyFund", "name": "Emergency fund", "rules": []any{emergencyFundRule}},
	}

	if activityCount > 0 {
		categories = append(categories,
			map[string]any{"key": "currencyClusterRisk", "name": "Currency cluster risk", "rules": []any{ccBaseCurrencyRule, ccCurrentRule}},
			map[string]any{"key": "assetClassClusterRisk", "name": "Asset class cluster risk", "rules": []any{equityRule, fixedIncomeRule}},
			map[string]any{"key": "accountClusterRisk", "name": "Account cluster risk", "rules": []any{accountCurrentRule, accountSingleRule}},
			map[string]any{"key": "economicMarketClusterRisk", "name": "Economic market cluster risk", "rules": []any{econDevRule, econEmgRule}},
			map[string]any{"key": "regionalMarketClusterRisk", "name": "Regional market cluster risk", "rules": []any{regAPRule, regEmgRule, regEuRule, regJpRule, regNARule}},
		)
	} else {
		categories = append(categories,
			map[string]any{"key": "currencyClusterRisk", "name": "Currency cluster risk"},
			map[string]any{"key": "assetClassClusterRisk", "name": "Asset class cluster risk"},
			map[string]any{"key": "accountClusterRisk", "name": "Account cluster risk"},
			map[string]any{"key": "economicMarketClusterRisk", "name": "Economic market cluster risk"},
			map[string]any{"key": "regionalMarketClusterRisk", "name": "Regional market cluster risk"},
		)
	}

	categories = append(categories,
		map[string]any{"key": "fees", "name": "Fees", "rules": []any{feeInitialRule, feeVolumeRule}},
	)

	// Compute statistics
	allRules := []map[string]any{
		buyingPowerRule, emergencyFundRule,
		ccBaseCurrencyRule, ccCurrentRule,
		equityRule, fixedIncomeRule,
		accountCurrentRule, accountSingleRule,
		econDevRule, econEmgRule,
		regAPRule, regEmgRule, regEuRule, regJpRule, regNARule,
		feeInitialRule, feeVolumeRule,
	}
	rulesActive := 0
	rulesFulfilled := 0
	for _, r := range allRules {
		if active, ok := r["isActive"].(bool); ok && active {
			rulesActive++
			if val, ok := r["value"].(bool); ok && val {
				rulesFulfilled++
			}
		}
	}

	return map[string]any{
		"xRay": map[string]any{
			"categories": categories,
			"statistics": map[string]any{
				"rulesActiveCount":    rulesActive,
				"rulesFulfilledCount": rulesFulfilled,
			},
		},
	}, nil
}

func tagsToAny(tags []calculator.Tag) []any {
	if len(tags) == 0 {
		return []any{}
	}
	result := make([]any, len(tags))
	for i, t := range tags {
		result[i] = map[string]string{"id": t.ID, "name": t.Name}
	}
	return result
}

// ensureJSONArray returns the raw JSON if non-nil and non-empty, otherwise []byte("[]").
func ensureJSONArray(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("[]")
	}
	return raw
}

// ensureJSONObject returns the raw JSON if non-nil and non-empty, otherwise []byte("{}").
func ensureJSONObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}
