package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/calculator"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// PortfolioService orchestrates portfolio calculations.
type PortfolioService struct {
	db           *sqlx.DB
	exchangeRate *ExchangeRateService
	userService  *UserService
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
func (s *PortfolioService) buildCalculator(ctx context.Context, userID string) (*calculator.Calculator, string, error) {
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
	activities := make([]calculator.Activity, len(ordersWithProfile))
	for i, o := range ordersWithProfile {
		assetSubClass := ""
		if o.SPAssetSubClass != nil {
			assetSubClass = *o.SPAssetSubClass
		}
		spUserID := ""
		if o.SPUserID != nil {
			spUserID = *o.SPUserID
		}
		activities[i] = calculator.Activity{
			Date:              calculator.FormatDate(o.Date),
			Fee:               decimal.NewFromFloat(o.Fee),
			FeeInBaseCurrency: decimal.NewFromFloat(o.Fee), // Will be converted by calculator
			Quantity:          decimal.NewFromFloat(o.Quantity),
			UnitPrice:         decimal.NewFromFloat(o.UnitPrice),
			Type:              string(o.Type),
			Symbol:            o.SPSymbol,
			DataSource:        o.SPDataSource,
			Currency:          o.SPCurrency,
			AssetSubClass:     assetSubClass,
			UserID:            spUserID,
			Tags:              []calculator.Tag{},
		}
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
	endDate := calculator.EndOfDay(calculator.ParseDate(calculator.FormatDate(calc.GetStartDate())).AddDate(100, 0, 0)) // large enough end
	// Use a sensible end date
	endDate = calculator.EndOfDay(calculator.ParseDate(calculator.FormatDate(endDate)))

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

// GetHoldings returns portfolio holdings.
func (s *PortfolioService) GetHoldings(ctx context.Context, userID string, dateRange string) (*PortfolioHoldingsResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID)
	if err != nil {
		return nil, err
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
		allocation := 0.0
		if totalValue > 0 {
			allocation = pos.ValueInBaseCurrency.InexactFloat64() / totalValue
		}

		h := PortfolioHolding{
			ActivitiesCount:        pos.ActivitiesCount,
			AllocationInPercentage: allocation,
			Currency:               pos.Currency,
			DataSource:             pos.DataSource,
			DateOfFirstActivity:    pos.DateOfFirstActivity,
			Dividend:               pos.Dividend.InexactFloat64(),
			Investment:             pos.Investment.InexactFloat64(),
			MarketPrice:            pos.MarketPrice,
			Name:                   pos.Symbol, // Will be enhanced with profile data
			Quantity:               pos.Quantity.InexactFloat64(),
			Symbol:                 pos.Symbol,
			Countries:              []any{},
			Holdings:               []any{},
			Sectors:                []any{},
			Tags:                   tagsToAny(pos.Tags),
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
func (s *PortfolioService) GetDetails(ctx context.Context, userID string, dateRange string) (*PortfolioDetailsResponse, error) {
	holdingsResp, err := s.GetHoldings(ctx, userID, dateRange)
	if err != nil {
		return nil, err
	}

	holdingsMap := make(map[string]PortfolioHolding)
	for _, h := range holdingsResp.Holdings {
		holdingsMap[h.Symbol] = h
	}

	return &PortfolioDetailsResponse{
		Accounts: map[string]AccountDetail{},
		HasError: false,
		Holdings: holdingsMap,
	}, nil
}

// GetPerformance returns portfolio performance data.
func (s *PortfolioService) GetPerformance(ctx context.Context, userID string, dateRange string) (*PortfolioPerformanceResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID)
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
func (s *PortfolioService) GetDividends(ctx context.Context, userID string, dateRange string, groupBy string) (*PortfolioDividendsResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID)
	if err != nil {
		return nil, err
	}

	snapshot, err := s.computeSnapshot(ctx, calc, baseCurrency)
	if err != nil {
		return nil, err
	}

	// Get historical data within range
	start, end := calculator.GetIntervalFromDateRange(dateRange, calc.GetStartDate())

	// Build dividend items from the snapshot chart
	var dividends []InvestmentItemResponse

	// For dividends, we need to look at activities directly
	for _, a := range calc.GetTransactionPoints() {
		d := calculator.ParseDate(a.Date)
		if d.Before(start) || d.After(end) {
			continue
		}
		// Interest from the transaction point represents dividends
		if !a.Interest.IsZero() {
			dividends = append(dividends, InvestmentItemResponse{
				Date:       a.Date,
				Investment: a.Interest.InexactFloat64(),
			})
		}
	}

	_ = snapshot

	if dividends == nil {
		dividends = []InvestmentItemResponse{}
	}

	return &PortfolioDividendsResponse{Dividends: dividends}, nil
}

// GetInvestments returns portfolio investment timeline.
func (s *PortfolioService) GetInvestments(ctx context.Context, userID string, dateRange string, groupBy string) (*PortfolioInvestmentsResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID)
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
func (s *PortfolioService) GetHolding(ctx context.Context, userID string, dataSource string, symbol string) (*PortfolioHoldingDetailResponse, error) {
	calc, baseCurrency, err := s.buildCalculator(ctx, userID)
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

	resp := &PortfolioHoldingDetailResponse{
		ActivitiesCount:        found.ActivitiesCount,
		AveragePrice:           found.AveragePrice.InexactFloat64(),
		DateOfFirstActivity:    found.DateOfFirstActivity,
		DividendInBaseCurrency: found.DividendInBaseCurrency.InexactFloat64(),
		FeeInBaseCurrency:      found.FeeInBaseCurrency.InexactFloat64(),
		HistoricalData:         []map[string]any{},
		Investment:             found.Investment.InexactFloat64(),
		MarketPrice:            found.MarketPrice,
		Quantity:               found.Quantity.InexactFloat64(),
		Symbol:                 found.Symbol,
		Tags:                   tagsToAny(found.Tags),
		ValueInBaseCurrency:    found.ValueInBaseCurrency.InexactFloat64(),
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

// GetReport returns portfolio report rules analysis.
func (s *PortfolioService) GetReport(ctx context.Context, userID string) (map[string]any, error) {
	// Stub: rules engine is complex and low-priority
	return map[string]any{}, nil
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
