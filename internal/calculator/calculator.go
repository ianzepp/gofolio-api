package calculator

import (
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// Calculator is the ROAI portfolio calculator.
type Calculator struct {
	activities          []Activity
	accountBalanceItems []AccountBalanceItem
	currency            string
	startDate           time.Time
	endDate             time.Time
	transactionPoints   []TransactionPoint
	chartDates          []string // lazily cached sorted chart date keys
}

// NewCalculator creates a Calculator from activities and account balance items.
// Activities are sorted by date ascending. Transaction points are computed immediately.
func NewCalculator(activities []Activity, accountBalanceItems []AccountBalanceItem, currency string) *Calculator {
	c := &Calculator{
		currency:            currency,
		accountBalanceItems: accountBalanceItems,
	}

	dateOfFirstActivity := time.Now().UTC()

	if len(accountBalanceItems) > 0 {
		d := ParseDate(accountBalanceItems[0].Date)
		if d.Before(dateOfFirstActivity) {
			dateOfFirstActivity = d
		}
	}

	// Map and sort activities
	now := time.Now().UTC()
	endOfToday := EndOfDay(now)

	mapped := make([]Activity, len(activities))
	for i, a := range activities {
		d := ParseDate(a.Date)
		if d.Before(dateOfFirstActivity) {
			dateOfFirstActivity = d
		}
		// Adapt future dates to today
		if d.After(now) {
			a.Date = FormatDate(endOfToday)
		}
		if a.Tags == nil {
			a.Tags = []Tag{}
		}
		mapped[i] = a
	}

	sort.Slice(mapped, func(i, j int) bool {
		return mapped[i].Date < mapped[j].Date
	})

	c.activities = mapped

	// Set date range: startDate = dateOfFirstActivity - 1 day, endDate = end of today
	start, _ := GetIntervalFromDateRange("max", StartOfDay(dateOfFirstActivity).AddDate(0, 0, -1))
	c.startDate = StartOfDay(start)
	c.endDate = EndOfDay(time.Now().UTC())

	c.computeTransactionPoints()

	return c
}

// GetTransactionPoints returns the computed transaction points.
func (c *Calculator) GetTransactionPoints() []TransactionPoint {
	return c.transactionPoints
}

// GetStartDate returns the calculated start date.
func (c *Calculator) GetStartDate() time.Time {
	return c.startDate
}

// GetInvestments returns per-transaction-point investment sums.
func (c *Calculator) GetInvestments() []InvestmentItem {
	if len(c.transactionPoints) == 0 {
		return nil
	}

	items := make([]InvestmentItem, len(c.transactionPoints))
	for i, tp := range c.transactionPoints {
		sum := decimal.Zero
		for _, item := range tp.Items {
			sum = sum.Add(item.Investment)
		}
		items[i] = InvestmentItem{
			Date:       tp.Date,
			Investment: sum.InexactFloat64(),
		}
	}
	return items
}

// GetInvestmentsByGroup groups historical investment data by month or year.
func (c *Calculator) GetInvestmentsByGroup(data []HistoricalDataItem, groupBy string) []InvestmentItem {
	grouped := map[string]decimal.Decimal{}

	for _, d := range data {
		var dateGroup string
		if groupBy == "month" {
			if len(d.Date) >= 7 {
				dateGroup = d.Date[:7]
			}
		} else {
			if len(d.Date) >= 4 {
				dateGroup = d.Date[:4]
			}
		}
		prev := grouped[dateGroup]
		grouped[dateGroup] = prev.Add(decimal.NewFromFloat(d.InvestmentValueWithCurrencyEffect))
	}

	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	items := make([]InvestmentItem, len(keys))
	for i, k := range keys {
		var date string
		if groupBy == "month" {
			date = k + "-01"
		} else {
			date = k + "-01-01"
		}
		items[i] = InvestmentItem{
			Date:       date,
			Investment: grouped[k].InexactFloat64(),
		}
	}
	return items
}

// ComputeSnapshot orchestrates the full portfolio calculation.
// Port of base class computeSnapshot() lines 179–643.
func (c *Calculator) ComputeSnapshot(marketData MarketData, exchangeRates ExchangeRates) *PortfolioSnapshot {
	emptySnapshot := &PortfolioSnapshot{
		CreatedAt:      time.Now().UTC(),
		Errors:         []AssetIdentifier{},
		HistoricalData: []HistoricalDataItem{},
		Positions:      []TimelinePosition{},
	}

	lastTransactionPoint := lastTP(c.transactionPoints)

	// Filter transaction points before endDate
	var transactionPoints []TransactionPoint
	for _, tp := range c.transactionPoints {
		if ParseDate(tp.Date).Before(c.endDate) {
			transactionPoints = append(transactionPoints, tp)
		}
	}

	if len(transactionPoints) == 0 {
		return emptySnapshot
	}

	// Collect symbols and currencies from last transaction point
	currencies := map[string]string{}
	type dgItem struct{ DataSource, Symbol string }
	var dataGatheringItems []dgItem

	lastTP := lastTransactionPoint
	for _, item := range lastTP.Items {
		if item.AssetSubClass != "CASH" {
			dataGatheringItems = append(dataGatheringItems, dgItem{item.DataSource, item.Symbol})
		}
		currencies[item.Symbol] = item.Currency
	}

	// Find firstTransactionPoint at or before startDate
	firstIndex := len(transactionPoints)
	var firstTransactionPointFound bool
	for i := 0; i < len(transactionPoints); i++ {
		d := ParseDate(transactionPoints[i].Date)
		if !d.Before(c.startDate) && !firstTransactionPointFound {
			firstIndex = i
			firstTransactionPointFound = true
		}
	}

	endDateString := FormatDate(c.endDate)
	daysInMarket := int(c.endDate.Sub(c.startDate).Hours() / 24)
	if daysInMarket < 1 {
		daysInMarket = 1
	}

	maxChartItems := 1000
	step := int(math.Round(float64(daysInMarket) / math.Min(float64(daysInMarket), float64(maxChartItems))))
	if step < 1 {
		step = 1
	}

	chartDateMap := c.getChartDateMap(c.startDate, c.endDate, step)

	// Add account balance dates
	for _, ab := range c.accountBalanceItems {
		chartDateMap[ab.Date] = true
	}

	// Sort chart dates
	chartDates := make([]string, 0, len(chartDateMap))
	for d := range chartDateMap {
		chartDates = append(chartDates, d)
	}
	sort.Strings(chartDates)

	if firstIndex > 0 {
		firstIndex--
	}

	errors := []AssetIdentifier{}
	hasAnySymbolMetricsErrors := false

	type positionWithHolding struct {
		TimelinePosition
		includeInHoldings bool
	}
	var positions []positionWithHolding

	// Per-symbol value maps
	type symbolValueMaps struct {
		currentValues                                  map[string]decimal.Decimal
		currentValuesWithCurrencyEffect                map[string]decimal.Decimal
		investmentValuesAccumulated                    map[string]decimal.Decimal
		investmentValuesAccumulatedWithCurrencyEffect  map[string]decimal.Decimal
		investmentValuesWithCurrencyEffect             map[string]decimal.Decimal
		netPerformanceValues                           map[string]decimal.Decimal
		netPerformanceValuesWithCurrencyEffect         map[string]decimal.Decimal
		timeWeightedInvestmentValues                   map[string]decimal.Decimal
		timeWeightedInvestmentValuesWithCurrencyEffect map[string]decimal.Decimal
	}
	valuesBySymbol := map[string]symbolValueMaps{}

	totalInterestWithCurrencyEffect := decimal.Zero
	totalLiabilitiesWithCurrencyEffect := decimal.Zero

	for _, item := range lastTP.Items {
		// Compute market price in base currency at end date
		symbolPrice := item.AveragePrice
		if mp, ok := marketData[endDateString]; ok {
			if p, ok2 := mp[item.Symbol]; ok2 {
				symbolPrice = p
			}
		}

		fxPair := item.Currency + c.currency
		fxRate := 1.0
		if rates, ok := exchangeRates[fxPair]; ok {
			if r, ok2 := rates[endDateString]; ok2 {
				fxRate = r
			}
		}

		marketPriceInBaseCurrency := symbolPrice.Mul(decimal.NewFromFloat(fxRate))

		// Get exchange rates for this symbol's currency
		var symbolExchangeRates map[string]float64
		if rates, ok := exchangeRates[fxPair]; ok {
			symbolExchangeRates = rates
		} else {
			symbolExchangeRates = map[string]float64{}
		}

		metrics := c.getSymbolMetrics(
			item.Symbol,
			item.DataSource,
			c.startDate,
			c.endDate,
			chartDateMap,
			marketData,
			symbolExchangeRates,
		)

		hasAnySymbolMetricsErrors = hasAnySymbolMetricsErrors || metrics.HasErrors

		includeInTotalAssetValue := item.AssetSubClass != "CASH"

		if includeInTotalAssetValue {
			valuesBySymbol[item.Symbol] = symbolValueMaps{
				currentValues:                                  metrics.CurrentValues,
				currentValuesWithCurrencyEffect:                metrics.CurrentValuesWithCurrencyEffect,
				investmentValuesAccumulated:                    metrics.InvestmentValuesAccumulated,
				investmentValuesAccumulatedWithCurrencyEffect:  metrics.InvestmentValuesAccumulatedWithCurrencyEffect,
				investmentValuesWithCurrencyEffect:             metrics.InvestmentValuesWithCurrencyEffect,
				netPerformanceValues:                           metrics.NetPerformanceValues,
				netPerformanceValuesWithCurrencyEffect:         metrics.NetPerformanceValuesWithCurrencyEffect,
				timeWeightedInvestmentValues:                   metrics.TimeWeightedInvestmentValues,
				timeWeightedInvestmentValuesWithCurrencyEffect: metrics.TimeWeightedInvestmentValuesWithCurrencyEffect,
			}
		}

		marketPriceFloat := 1.0
		if mp, ok := marketData[endDateString]; ok {
			if p, ok2 := mp[item.Symbol]; ok2 {
				marketPriceFloat = p.InexactFloat64()
			}
		}

		var grossPerf, grossPerfPct, grossPerfPctFX, grossPerfFX *decimal.Decimal
		var netPerf, netPerfPct *decimal.Decimal
		var netPerfPctFXMap, netPerfFXMap map[string]decimal.Decimal

		if !metrics.HasErrors {
			gp := metrics.GrossPerformance
			grossPerf = &gp
			gpp := metrics.GrossPerformancePercentage
			grossPerfPct = &gpp
			gppfx := metrics.GrossPerformancePercentageWithCurrencyEffect
			grossPerfPctFX = &gppfx
			gpfx := metrics.GrossPerformanceWithCurrencyEffect
			grossPerfFX = &gpfx
			np := metrics.NetPerformance
			netPerf = &np
			npp := metrics.NetPerformancePercentage
			netPerfPct = &npp
			netPerfPctFXMap = metrics.NetPerformancePercentageWithCurrencyEffectMap
			netPerfFXMap = metrics.NetPerformanceWithCurrencyEffectMap
		}

		positions = append(positions, positionWithHolding{
			TimelinePosition: TimelinePosition{
				ActivitiesCount:            item.ActivitiesCount,
				AveragePrice:               item.AveragePrice,
				Currency:                   item.Currency,
				DataSource:                 item.DataSource,
				DateOfFirstActivity:        item.DateOfFirstActivity,
				Dividend:                   metrics.TotalDividend,
				DividendInBaseCurrency:     metrics.TotalDividendInBaseCurrency,
				Fee:                        item.Fee,
				FeeInBaseCurrency:          item.FeeInBaseCurrency,
				GrossPerformance:           grossPerf,
				GrossPerformancePercentage: grossPerfPct,
				GrossPerformancePercentageWithCurrencyEffect:  grossPerfPctFX,
				GrossPerformanceWithCurrencyEffect:            grossPerfFX,
				IncludeInHoldings:                             item.IncludeInHoldings,
				IncludeInTotalAssetValue:                      includeInTotalAssetValue,
				Investment:                                    metrics.TotalInvestment,
				InvestmentWithCurrencyEffect:                  metrics.TotalInvestmentWithCurrencyEffect,
				MarketPrice:                                   marketPriceFloat,
				MarketPriceInBaseCurrency:                     marketPriceInBaseCurrency.InexactFloat64(),
				NetPerformance:                                netPerf,
				NetPerformancePercentage:                      netPerfPct,
				NetPerformancePercentageWithCurrencyEffectMap: netPerfPctFXMap,
				NetPerformanceWithCurrencyEffectMap:           netPerfFXMap,
				Quantity:                                      item.Quantity,
				Symbol:                                        item.Symbol,
				Tags:                                          item.Tags,
				TimeWeightedInvestment:                        metrics.TimeWeightedInvestment,
				TimeWeightedInvestmentWithCurrencyEffect:      metrics.TimeWeightedInvestmentWithCurrencyEffect,
				ValueInBaseCurrency:                           marketPriceInBaseCurrency.Mul(item.Quantity),
			},
			includeInHoldings: item.IncludeInHoldings,
		})

		totalInterestWithCurrencyEffect = totalInterestWithCurrencyEffect.Add(metrics.TotalInterestInBaseCurrency)
		totalLiabilitiesWithCurrencyEffect = totalLiabilitiesWithCurrencyEffect.Add(metrics.TotalLiabilitiesInBaseCurrency)

		if (metrics.HasErrors) && item.Investment.GreaterThan(decimal.Zero) && !item.SkipErrors {
			errors = append(errors, AssetIdentifier{DataSource: item.DataSource, Symbol: item.Symbol})
		}
	}

	// Build account balance map (forward-fill)
	accountBalanceItemsMap := map[string]decimal.Decimal{}
	for _, ab := range c.accountBalanceItems {
		accountBalanceItemsMap[ab.Date] = ab.Value
	}

	accountBalanceMap := map[string]decimal.Decimal{}
	lastKnownBalance := decimal.Zero

	// Accumulate per-date totals
	type accumulatedValues struct {
		investmentValueWithCurrencyEffect                  decimal.Decimal
		totalAccountBalanceWithCurrencyEffect              decimal.Decimal
		totalCurrentValue                                  decimal.Decimal
		totalCurrentValueWithCurrencyEffect                decimal.Decimal
		totalInvestmentValue                               decimal.Decimal
		totalInvestmentValueWithCurrencyEffect             decimal.Decimal
		totalNetPerformanceValue                           decimal.Decimal
		totalNetPerformanceValueWithCurrencyEffect         decimal.Decimal
		totalTimeWeightedInvestmentValue                   decimal.Decimal
		totalTimeWeightedInvestmentValueWithCurrencyEffect decimal.Decimal
	}
	accumulatedValuesByDate := map[string]*accumulatedValues{}

	for _, dateString := range chartDates {
		if v, ok := accountBalanceItemsMap[dateString]; ok {
			lastKnownBalance = v
		}
		accountBalanceMap[dateString] = lastKnownBalance

		for sym, sv := range valuesBySymbol {
			_ = sym
			cv := getOrZero(sv.currentValues, dateString)
			cvFX := getOrZero(sv.currentValuesWithCurrencyEffect, dateString)
			ivAcc := getOrZero(sv.investmentValuesAccumulated, dateString)
			ivAccFX := getOrZero(sv.investmentValuesAccumulatedWithCurrencyEffect, dateString)
			ivFX := getOrZero(sv.investmentValuesWithCurrencyEffect, dateString)
			npv := getOrZero(sv.netPerformanceValues, dateString)
			npvFX := getOrZero(sv.netPerformanceValuesWithCurrencyEffect, dateString)
			twiv := getOrZero(sv.timeWeightedInvestmentValues, dateString)
			twivFX := getOrZero(sv.timeWeightedInvestmentValuesWithCurrencyEffect, dateString)

			acc := accumulatedValuesByDate[dateString]
			if acc == nil {
				acc = &accumulatedValues{}
				accumulatedValuesByDate[dateString] = acc
			}

			acc.investmentValueWithCurrencyEffect = acc.investmentValueWithCurrencyEffect.Add(ivFX)
			acc.totalAccountBalanceWithCurrencyEffect = accountBalanceMap[dateString]
			acc.totalCurrentValue = acc.totalCurrentValue.Add(cv)
			acc.totalCurrentValueWithCurrencyEffect = acc.totalCurrentValueWithCurrencyEffect.Add(cvFX)
			acc.totalInvestmentValue = acc.totalInvestmentValue.Add(ivAcc)
			acc.totalInvestmentValueWithCurrencyEffect = acc.totalInvestmentValueWithCurrencyEffect.Add(ivAccFX)
			acc.totalNetPerformanceValue = acc.totalNetPerformanceValue.Add(npv)
			acc.totalNetPerformanceValueWithCurrencyEffect = acc.totalNetPerformanceValueWithCurrencyEffect.Add(npvFX)
			acc.totalTimeWeightedInvestmentValue = acc.totalTimeWeightedInvestmentValue.Add(twiv)
			acc.totalTimeWeightedInvestmentValueWithCurrencyEffect = acc.totalTimeWeightedInvestmentValueWithCurrencyEffect.Add(twivFX)
		}
	}

	// Build historical data
	historicalData := make([]HistoricalDataItem, 0, len(chartDates))
	for _, dateString := range chartDates {
		acc := accumulatedValuesByDate[dateString]
		if acc == nil {
			acc = &accumulatedValues{
				totalAccountBalanceWithCurrencyEffect: accountBalanceMap[dateString],
			}
		}

		netPerfPct := 0.0
		if !acc.totalTimeWeightedInvestmentValue.IsZero() {
			netPerfPct = acc.totalNetPerformanceValue.Div(acc.totalTimeWeightedInvestmentValue).InexactFloat64()
		}

		netPerfPctFX := 0.0
		if !acc.totalTimeWeightedInvestmentValueWithCurrencyEffect.IsZero() {
			netPerfPctFX = acc.totalNetPerformanceValueWithCurrencyEffect.Div(acc.totalTimeWeightedInvestmentValueWithCurrencyEffect).InexactFloat64()
		}

		historicalData = append(historicalData, HistoricalDataItem{
			Date:                              dateString,
			InvestmentValueWithCurrencyEffect: acc.investmentValueWithCurrencyEffect.InexactFloat64(),
			NetPerformance:                    acc.totalNetPerformanceValue.InexactFloat64(),
			NetPerformanceInPercentage:        netPerfPct,
			NetPerformanceInPercentageWithCurrencyEffect: netPerfPctFX,
			NetPerformanceWithCurrencyEffect:             acc.totalNetPerformanceValueWithCurrencyEffect.InexactFloat64(),
			NetWorth:                                     acc.totalCurrentValueWithCurrencyEffect.Add(acc.totalAccountBalanceWithCurrencyEffect).InexactFloat64(),
			TotalAccountBalance:                          acc.totalAccountBalanceWithCurrencyEffect.InexactFloat64(),
			TotalInvestment:                              acc.totalInvestmentValue.InexactFloat64(),
			TotalInvestmentValueWithCurrencyEffect:       acc.totalInvestmentValueWithCurrencyEffect.InexactFloat64(),
			Value:                                        acc.totalCurrentValue.InexactFloat64(),
			ValueWithCurrencyEffect:                      acc.totalCurrentValueWithCurrencyEffect.InexactFloat64(),
		})
	}

	// Calculate overall performance
	tlPositions := make([]TimelinePosition, len(positions))
	for i, p := range positions {
		tlPositions[i] = p.TimelinePosition
	}
	overall := c.calculateOverallPerformance(tlPositions)

	// Filter positions to includeInHoldings
	var filteredPositions []TimelinePosition
	for _, p := range positions {
		if p.includeInHoldings {
			filteredPositions = append(filteredPositions, p.TimelinePosition)
		}
	}
	if filteredPositions == nil {
		filteredPositions = []TimelinePosition{}
	}

	return &PortfolioSnapshot{
		ActivitiesCount:                    overall.ActivitiesCount,
		CreatedAt:                          time.Now().UTC(),
		CurrentValueInBaseCurrency:         overall.CurrentValueInBaseCurrency,
		Errors:                             errors,
		HasErrors:                          hasAnySymbolMetricsErrors || overall.HasErrors,
		HistoricalData:                     historicalData,
		Positions:                          filteredPositions,
		TotalFeesWithCurrencyEffect:        overall.TotalFeesWithCurrencyEffect,
		TotalInterestWithCurrencyEffect:    totalInterestWithCurrencyEffect,
		TotalInvestment:                    overall.TotalInvestment,
		TotalInvestmentWithCurrencyEffect:  overall.TotalInvestmentWithCurrencyEffect,
		TotalLiabilitiesWithCurrencyEffect: totalLiabilitiesWithCurrencyEffect,
	}
}

// GetPerformance returns chart data filtered to a date range.
func (c *Calculator) GetPerformance(start, end time.Time, snapshot *PortfolioSnapshot) []HistoricalDataItem {
	var chart []HistoricalDataItem
	var netPerfAtStart *float64
	var netPerfFXAtStart *float64
	var totalInvestmentValues []float64

	for _, item := range snapshot.HistoricalData {
		d := ParseDate(item.Date)
		if d.Before(start) || d.After(end) {
			continue
		}

		if netPerfAtStart == nil {
			v := item.NetPerformance
			netPerfAtStart = &v
			vfx := item.NetPerformanceWithCurrencyEffect
			netPerfFXAtStart = &vfx
		}

		npSinceStart := item.NetPerformance - *netPerfAtStart
		npFXSinceStart := item.NetPerformanceWithCurrencyEffect - *netPerfFXAtStart

		if item.TotalInvestmentValueWithCurrencyEffect > 0 {
			totalInvestmentValues = append(totalInvestmentValues, item.TotalInvestmentValueWithCurrencyEffect)
		}

		twi := 0.0
		if len(totalInvestmentValues) > 0 {
			sum := 0.0
			for _, v := range totalInvestmentValues {
				sum += v
			}
			twi = sum / float64(len(totalInvestmentValues))
		}

		npPct := 0.0
		if twi != 0 {
			npPct = npSinceStart / twi
		}
		npPctFX := 0.0
		if twi != 0 {
			npPctFX = npFXSinceStart / twi
		}

		chart = append(chart, HistoricalDataItem{
			Date:                              item.Date,
			InvestmentValueWithCurrencyEffect: item.InvestmentValueWithCurrencyEffect,
			NetPerformance:                    npSinceStart,
			NetPerformanceInPercentage:        npPct,
			NetPerformanceInPercentageWithCurrencyEffect: npPctFX,
			NetPerformanceWithCurrencyEffect:             npFXSinceStart,
			NetWorth:                                     item.NetWorth,
			TotalAccountBalance:                          item.TotalAccountBalance,
			TotalInvestment:                              item.TotalInvestment,
			TotalInvestmentValueWithCurrencyEffect:       item.TotalInvestmentValueWithCurrencyEffect,
			Value:                                        item.Value,
			ValueWithCurrencyEffect:                      item.ValueWithCurrencyEffect,
		})
	}

	if chart == nil {
		chart = []HistoricalDataItem{}
	}
	return chart
}

// computeTransactionPoints builds the cumulative transaction point timeline.
// Port of base class lines 927–1088.
func (c *Calculator) computeTransactionPoints() {
	c.transactionPoints = nil
	symbols := map[string]TransactionPointSymbol{}

	var lastDate string
	var lastTP *TransactionPoint

	for _, a := range c.activities {
		var currentItem TransactionPointSymbol

		factor := GetFactor(a.Type)
		skipErrors := a.UserID != ""
		factorDec := decimal.NewFromInt(int64(factor))

		old, exists := symbols[a.Symbol]

		if exists {
			investment := old.Investment
			newQuantity := a.Quantity.Mul(factorDec).Add(old.Quantity)

			if a.Type == "BUY" {
				if old.Investment.GreaterThanOrEqual(decimal.Zero) {
					investment = old.Investment.Add(a.Quantity.Mul(a.UnitPrice))
				} else {
					investment = old.Investment.Add(a.Quantity.Mul(old.AveragePrice))
				}
			} else if a.Type == "SELL" {
				if old.Investment.GreaterThan(decimal.Zero) {
					investment = old.Investment.Sub(a.Quantity.Mul(old.AveragePrice))
				} else {
					investment = old.Investment.Sub(a.Quantity.Mul(a.UnitPrice))
				}
			}

			epsilon := decimal.NewFromFloat(math.SmallestNonzeroFloat64 * 1e10)
			if newQuantity.Abs().LessThan(epsilon) {
				investment = decimal.Zero
				newQuantity = decimal.Zero
			}

			avgPrice := decimal.Zero
			if !newQuantity.IsZero() {
				avgPrice = investment.Div(newQuantity).Abs()
			}

			// Merge tags
			mergedTags := append(old.Tags, a.Tags...)
			mergedTags = uniqueTags(mergedTags)

			currentItem = TransactionPointSymbol{
				Symbol:              a.Symbol,
				DataSource:          a.DataSource,
				Currency:            a.Currency,
				AssetSubClass:       a.AssetSubClass,
				Quantity:            newQuantity,
				Investment:          investment,
				AveragePrice:        avgPrice,
				Fee:                 old.Fee.Add(a.Fee),
				FeeInBaseCurrency:   old.FeeInBaseCurrency.Add(a.FeeInBaseCurrency),
				Dividend:            decimal.Zero,
				ActivitiesCount:     old.ActivitiesCount + 1,
				DateOfFirstActivity: old.DateOfFirstActivity,
				IncludeInHoldings:   old.IncludeInHoldings,
				SkipErrors:          skipErrors,
				Tags:                mergedTags,
			}
		} else {
			tags := a.Tags
			if tags == nil {
				tags = []Tag{}
			}
			currentItem = TransactionPointSymbol{
				Symbol:              a.Symbol,
				DataSource:          a.DataSource,
				Currency:            a.Currency,
				AssetSubClass:       a.AssetSubClass,
				Quantity:            a.Quantity.Mul(factorDec),
				Investment:          a.UnitPrice.Mul(a.Quantity).Mul(factorDec),
				AveragePrice:        a.UnitPrice,
				Fee:                 a.Fee,
				FeeInBaseCurrency:   a.FeeInBaseCurrency,
				Dividend:            decimal.Zero,
				ActivitiesCount:     1,
				DateOfFirstActivity: a.Date,
				IncludeInHoldings:   IsInvestmentActivity(a.Type),
				SkipErrors:          skipErrors,
				Tags:                tags,
			}
		}

		symbols[a.Symbol] = currentItem

		// Build items list: all symbols except current, then add current, sort by symbol
		var prevItems []TransactionPointSymbol
		if lastTP != nil {
			prevItems = lastTP.Items
		}

		newItems := make([]TransactionPointSymbol, 0, len(prevItems)+1)
		for _, item := range prevItems {
			if item.Symbol != a.Symbol {
				newItems = append(newItems, item)
			}
		}
		newItems = append(newItems, currentItem)
		sort.Slice(newItems, func(i, j int) bool {
			return newItems[i].Symbol < newItems[j].Symbol
		})

		fees := decimal.Zero
		if a.Type == "FEE" {
			fees = a.Fee
		}

		interest := decimal.Zero
		if a.Type == "INTEREST" {
			interest = a.Quantity.Mul(a.UnitPrice)
		}

		liabilities := decimal.Zero
		if a.Type == "LIABILITY" {
			liabilities = a.Quantity.Mul(a.UnitPrice)
		}

		if lastDate != a.Date || lastTP == nil {
			tp := TransactionPoint{
				Date:        a.Date,
				Items:       newItems,
				Fees:        fees,
				Interest:    interest,
				Liabilities: liabilities,
			}
			c.transactionPoints = append(c.transactionPoints, tp)
			lastTP = &c.transactionPoints[len(c.transactionPoints)-1]
		} else {
			lastTP.Fees = lastTP.Fees.Add(fees)
			lastTP.Interest = lastTP.Interest.Add(interest)
			lastTP.Items = newItems
			lastTP.Liabilities = lastTP.Liabilities.Add(liabilities)
		}

		lastDate = a.Date
	}
}

// getChartDateMap creates a set of dates to include in the chart.
// Port of base class lines 840–924.
func (c *Calculator) getChartDateMap(startDate, endDate time.Time, step int) map[string]bool {
	chartDateMap := map[string]bool{}

	// 1. Transaction point dates
	for _, tp := range c.transactionPoints {
		chartDateMap[tp.Date] = true
	}

	// 2. Every Nth day from start to end
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, step) {
		chartDateMap[FormatDate(d)] = true
	}

	if step > 1 {
		// Last 90 days at step=3
		start90 := endDate.AddDate(0, 0, -90)
		for d := start90; !d.After(endDate); d = d.AddDate(0, 0, 3) {
			chartDateMap[FormatDate(d)] = true
		}

		// Last 30 days at step=1
		start30 := endDate.AddDate(0, 0, -30)
		for d := start30; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			chartDateMap[FormatDate(d)] = true
		}
	}

	// End date always present
	chartDateMap[FormatDate(endDate)] = true

	// Key date ranges
	for _, dr := range []string{"1d", "1y", "5y", "max", "mtd", "wtd", "ytd"} {
		drStart, drEnd := GetIntervalFromDateRange(dr)
		ds := FormatDate(drStart)
		de := FormatDate(drEnd)

		if !drStart.Before(startDate) && !drStart.After(endDate) {
			chartDateMap[ds] = true
		}
		if !drEnd.Before(startDate) && !drEnd.After(endDate) {
			chartDateMap[de] = true
		}
	}

	// Calendar year boundaries
	for y := startDate.Year(); y <= endDate.Year(); y++ {
		yearStart := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		yearEnd := time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC)

		if !yearStart.Before(startDate) && !yearStart.After(endDate) {
			chartDateMap[FormatDate(yearStart)] = true
		}
		if !yearEnd.Before(startDate) && !yearEnd.After(endDate) {
			chartDateMap[FormatDate(yearEnd)] = true
		}
	}

	return chartDateMap
}

// Helper functions

func lastTP(tps []TransactionPoint) *TransactionPoint {
	if len(tps) == 0 {
		return nil
	}
	return &tps[len(tps)-1]
}

func getOrZero(m map[string]decimal.Decimal, key string) decimal.Decimal {
	if v, ok := m[key]; ok {
		return v
	}
	return decimal.Zero
}

func uniqueTags(tags []Tag) []Tag {
	seen := map[string]bool{}
	result := make([]Tag, 0, len(tags))
	for _, t := range tags {
		if !seen[t.ID] {
			seen[t.ID] = true
			result = append(result, t)
		}
	}
	return result
}
