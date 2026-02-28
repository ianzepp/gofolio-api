package calculator

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// getSymbolMetrics computes all metrics for a single symbol.
// Port of ROAI lines 130–1009.
func (c *Calculator) getSymbolMetrics(
	symbol string,
	dataSource string,
	start, end time.Time,
	chartDateMap map[string]bool,
	marketSymbolMap MarketData,
	exchangeRates map[string]float64,
) SymbolMetrics {
	todayStr := FormatDate(time.Now().UTC())
	currentExchangeRate := exchangeRates[todayStr]

	emptyMetrics := SymbolMetrics{
		CurrentValues:                                  map[string]decimal.Decimal{},
		CurrentValuesWithCurrencyEffect:                map[string]decimal.Decimal{},
		FeesWithCurrencyEffect:                         decimal.Zero,
		GrossPerformance:                               decimal.Zero,
		GrossPerformancePercentage:                     decimal.Zero,
		GrossPerformancePercentageWithCurrencyEffect:   decimal.Zero,
		GrossPerformanceWithCurrencyEffect:             decimal.Zero,
		HasErrors:                                      false,
		InitialValue:                                   decimal.Zero,
		InitialValueWithCurrencyEffect:                 decimal.Zero,
		InvestmentValuesAccumulated:                    map[string]decimal.Decimal{},
		InvestmentValuesAccumulatedWithCurrencyEffect:  map[string]decimal.Decimal{},
		InvestmentValuesWithCurrencyEffect:             map[string]decimal.Decimal{},
		NetPerformance:                                 decimal.Zero,
		NetPerformancePercentage:                       decimal.Zero,
		NetPerformancePercentageWithCurrencyEffectMap:  map[string]decimal.Decimal{},
		NetPerformanceValues:                           map[string]decimal.Decimal{},
		NetPerformanceValuesWithCurrencyEffect:         map[string]decimal.Decimal{},
		NetPerformanceWithCurrencyEffectMap:            map[string]decimal.Decimal{},
		TimeWeightedInvestment:                         decimal.Zero,
		TimeWeightedInvestmentValues:                   map[string]decimal.Decimal{},
		TimeWeightedInvestmentValuesWithCurrencyEffect: map[string]decimal.Decimal{},
		TimeWeightedInvestmentWithCurrencyEffect:       decimal.Zero,
		TotalAccountBalanceInBaseCurrency:              decimal.Zero,
		TotalDividend:                                  decimal.Zero,
		TotalDividendInBaseCurrency:                    decimal.Zero,
		TotalInterest:                                  decimal.Zero,
		TotalInterestInBaseCurrency:                    decimal.Zero,
		TotalInvestment:                                decimal.Zero,
		TotalInvestmentWithCurrencyEffect:              decimal.Zero,
		TotalLiabilities:                               decimal.Zero,
		TotalLiabilitiesInBaseCurrency:                 decimal.Zero,
	}

	// Legacy parity: filter by symbol only (not dataSource+symbol), matching the
	// NestJS ROAI implementation. This can blend same-symbol activities across
	// sources, but is preserved to keep response compatibility.
	// Filter orders for this symbol
	var orders []orderItem
	for _, a := range c.activities {
		if a.Symbol == symbol {
			orders = append(orders, orderItem{Activity: a})
		}
	}

	if len(orders) == 0 {
		return emptyMetrics
	}

	isCash := orders[0].AssetSubClass == "CASH"

	endDateString := FormatDate(end)
	startDateString := FormatDate(start)

	// Price validation
	unitPriceAtStartDate := getMarketPrice(marketSymbolMap, startDateString, symbol)
	unitPriceAtEndDate := getMarketPrice(marketSymbolMap, endDateString, symbol)

	latestActivity := orders[len(orders)-1]
	if dataSource == "MANUAL" &&
		(latestActivity.Type == "BUY" || latestActivity.Type == "SELL") &&
		!latestActivity.UnitPrice.IsZero() &&
		unitPriceAtEndDate == nil {
		p := latestActivity.UnitPrice
		unitPriceAtEndDate = &p
	} else if isCash {
		one := decimal.NewFromInt(1)
		unitPriceAtEndDate = &one
	}

	dateOfFirstTransaction := ParseDate(orders[0].Date)

	if unitPriceAtEndDate == nil ||
		(unitPriceAtStartDate == nil && dateOfFirstTransaction.Before(start)) {
		errMetrics := emptyMetrics
		errMetrics.HasErrors = true
		return errMetrics
	}

	// Add synthetic start and end orders
	startOrder := orderItem{
		Activity: Activity{
			Date:              startDateString,
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.Zero,
			Type:              "BUY",
			Symbol:            symbol,
			DataSource:        dataSource,
			Tags:              []Tag{},
		},
		ItemType: "start",
	}
	if unitPriceAtStartDate != nil {
		startOrder.UnitPrice = *unitPriceAtStartDate
	}
	if isCash {
		startOrder.AssetSubClass = "CASH"
	}

	endOrder := orderItem{
		Activity: Activity{
			Date:              endDateString,
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.Zero,
			UnitPrice:         *unitPriceAtEndDate,
			Type:              "BUY",
			Symbol:            symbol,
			DataSource:        dataSource,
			Tags:              []Tag{},
		},
		ItemType: "end",
	}
	if isCash {
		endOrder.AssetSubClass = "CASH"
	}

	orders = append(orders, startOrder, endOrder)

	// Build ordersByDate map
	ordersByDate := map[string][]int{}
	for i := range orders {
		ordersByDate[orders[i].Date] = append(ordersByDate[orders[i].Date], i)
	}

	// Cache chart dates
	if c.chartDates == nil {
		keys := make([]string, 0, len(chartDateMap))
		for k := range chartDateMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		c.chartDates = keys
	}

	var lastUnitPrice *decimal.Decimal

	// Fill chart dates with market data or filler orders
	for _, dateString := range c.chartDates {
		if dateString < startDateString {
			continue
		}
		if dateString > endDateString {
			break
		}

		if indices, ok := ordersByDate[dateString]; ok && len(indices) > 0 {
			for _, idx := range indices {
				mp := getMarketPrice(marketSymbolMap, dateString, symbol)
				if mp != nil {
					orders[idx].UnitPriceFromMarketData = mp
				} else if lastUnitPrice != nil {
					p := *lastUnitPrice
					orders[idx].UnitPriceFromMarketData = &p
				}
			}
		} else {
			// Insert filler order
			mp := getMarketPrice(marketSymbolMap, dateString, symbol)
			var price decimal.Decimal
			if mp != nil {
				price = *mp
			} else if lastUnitPrice != nil {
				price = *lastUnitPrice
			}
			filler := orderItem{
				Activity: Activity{
					Date:              dateString,
					Fee:               decimal.Zero,
					FeeInBaseCurrency: decimal.Zero,
					Quantity:          decimal.Zero,
					UnitPrice:         price,
					Type:              "BUY",
					Symbol:            symbol,
					DataSource:        dataSource,
					Tags:              []Tag{},
				},
			}
			if isCash {
				filler.AssetSubClass = "CASH"
			}
			fp := price
			filler.UnitPriceFromMarketData = &fp
			orders = append(orders, filler)
		}

		latestOrder := orders[len(orders)-1]
		if latestOrder.UnitPriceFromMarketData != nil {
			p := *latestOrder.UnitPriceFromMarketData
			lastUnitPrice = &p
		} else if !latestOrder.UnitPrice.IsZero() {
			p := latestOrder.UnitPrice
			lastUnitPrice = &p
		}
	}

	// Sort orders: start orders sort -1ms before their date, end orders +1ms after
	sort.SliceStable(orders, func(i, j int) bool {
		di := orders[i].Date
		dj := orders[j].Date
		if di != dj {
			return di < dj
		}
		return itemTypeSortKey(orders[i].ItemType) < itemTypeSortKey(orders[j].ItemType)
	})

	// Find indices of start and end orders
	indexOfStartOrder := -1
	indexOfEndOrder := -1
	for i := range orders {
		if orders[i].ItemType == "start" && indexOfStartOrder < 0 {
			indexOfStartOrder = i
		}
		if orders[i].ItemType == "end" {
			indexOfEndOrder = i
		}
	}

	// Main calculation loop
	currentValues := map[string]decimal.Decimal{}
	currentValuesWithCurrencyEffect := map[string]decimal.Decimal{}
	investmentValuesAccumulated := map[string]decimal.Decimal{}
	investmentValuesAccumulatedWithCurrencyEffect := map[string]decimal.Decimal{}
	investmentValuesWithCurrencyEffect := map[string]decimal.Decimal{}
	netPerformanceValues := map[string]decimal.Decimal{}
	netPerformanceValuesWithCurrencyEffect := map[string]decimal.Decimal{}
	timeWeightedInvestmentValues := map[string]decimal.Decimal{}
	timeWeightedInvestmentValuesWithCurrencyEffect := map[string]decimal.Decimal{}

	fees := decimal.Zero
	feesAtStartDate := decimal.Zero
	feesAtStartDateWithCurrencyEffect := decimal.Zero
	feesWithCurrencyEffect := decimal.Zero
	grossPerformance := decimal.Zero
	grossPerformanceWithCurrencyEffect := decimal.Zero
	grossPerformanceAtStartDate := decimal.Zero
	grossPerformanceAtStartDateWithCurrencyEffect := decimal.Zero
	grossPerformanceFromSells := decimal.Zero
	grossPerformanceFromSellsWithCurrencyEffect := decimal.Zero

	var initialValue *decimal.Decimal
	var initialValueWithCurrencyEffect *decimal.Decimal
	var investmentAtStartDate *decimal.Decimal
	var investmentAtStartDateWithCurrencyEffect *decimal.Decimal
	var valueAtStartDate decimal.Decimal
	var valueAtStartDateWithCurrencyEffect decimal.Decimal

	lastAveragePrice := decimal.Zero
	lastAveragePriceWithCurrencyEffect := decimal.Zero

	totalDividend := decimal.Zero
	totalDividendInBaseCurrency := decimal.Zero
	totalInterest := decimal.Zero
	totalInterestInBaseCurrency := decimal.Zero
	totalInvestment := decimal.Zero
	totalInvestmentFromBuyTransactions := decimal.Zero
	totalInvestmentFromBuyTransactionsWithCurrencyEffect := decimal.Zero
	totalInvestmentWithCurrencyEffect := decimal.Zero
	totalLiabilities := decimal.Zero
	totalLiabilitiesInBaseCurrency := decimal.Zero
	totalQuantityFromBuyTransactions := decimal.Zero
	totalUnits := decimal.Zero

	totalInvestmentDays := 0.0
	sumOfTimeWeightedInvestments := decimal.Zero
	sumOfTimeWeightedInvestmentsWithCurrencyEffect := decimal.Zero

	for i := 0; i < len(orders); i++ {
		order := &orders[i]

		exchangeRateAtOrderDate := exchangeRates[order.Date]

		// Dividend/Interest/Liability accumulation
		if order.Type == "DIVIDEND" {
			dividend := order.Quantity.Mul(order.UnitPrice)
			totalDividend = totalDividend.Add(dividend)
			totalDividendInBaseCurrency = totalDividendInBaseCurrency.Add(
				dividend.Mul(decFromFloat(exchangeRateAtOrderDate, 1.0)),
			)
		} else if order.Type == "INTEREST" {
			interest := order.Quantity.Mul(order.UnitPrice)
			totalInterest = totalInterest.Add(interest)
			totalInterestInBaseCurrency = totalInterestInBaseCurrency.Add(
				interest.Mul(decFromFloat(exchangeRateAtOrderDate, 1.0)),
			)
		} else if order.Type == "LIABILITY" {
			liability := order.Quantity.Mul(order.UnitPrice)
			totalLiabilities = totalLiabilities.Add(liability)
			totalLiabilitiesInBaseCurrency = totalLiabilitiesInBaseCurrency.Add(
				liability.Mul(decFromFloat(exchangeRateAtOrderDate, 1.0)),
			)
		}

		// Handle start order unit price
		if order.ItemType == "start" {
			if indexOfStartOrder == 0 && i+1 < len(orders) {
				order.UnitPrice = orders[i+1].UnitPrice
			} else if unitPriceAtStartDate != nil {
				order.UnitPrice = *unitPriceAtStartDate
			}
		}

		// Convert fees to base currency
		if !order.Fee.IsZero() {
			feeInBase := order.Fee.Mul(decFromFloat(currentExchangeRate, 1.0))
			order.FeeInBaseCurrencyCalc = feeInBase
			order.FeeInBaseCurrencyWithCurrencyEffect = order.Fee.Mul(decFromFloat(exchangeRateAtOrderDate, 1.0))
		}

		// Compute unit prices in base currency
		unitPrice := order.UnitPrice
		if order.Type == "BUY" || order.Type == "SELL" {
			// use order.UnitPrice as-is
		} else if order.UnitPriceFromMarketData != nil {
			unitPrice = *order.UnitPriceFromMarketData
		}

		if !unitPrice.IsZero() {
			upBase := unitPrice.Mul(decFromFloat(currentExchangeRate, 1.0))
			order.UnitPriceInBaseCurrency = &upBase
			upBaseFX := unitPrice.Mul(decFromFloat(exchangeRateAtOrderDate, 1.0))
			order.UnitPriceInBaseCurrencyWithCurrencyEffect = &upBaseFX
		}

		// Market price in base currency
		var marketPriceInBaseCurrency decimal.Decimal
		var marketPriceInBaseCurrencyWithCurrencyEffect decimal.Decimal
		if order.UnitPriceFromMarketData != nil {
			marketPriceInBaseCurrency = order.UnitPriceFromMarketData.Mul(decFromFloat(currentExchangeRate, 1.0))
			marketPriceInBaseCurrencyWithCurrencyEffect = order.UnitPriceFromMarketData.Mul(decFromFloat(exchangeRateAtOrderDate, 1.0))
		}

		valueOfInvestmentBeforeTransaction := totalUnits.Mul(marketPriceInBaseCurrency)
		valueOfInvestmentBeforeTransactionWithCurrencyEffect := totalUnits.Mul(marketPriceInBaseCurrencyWithCurrencyEffect)

		// Capture investment at start date
		if investmentAtStartDate == nil && i >= indexOfStartOrder {
			ti := totalInvestment
			investmentAtStartDate = &ti
			tiFX := totalInvestmentWithCurrencyEffect
			investmentAtStartDateWithCurrencyEffect = &tiFX
			valueAtStartDate = valueOfInvestmentBeforeTransaction
			valueAtStartDateWithCurrencyEffect = valueOfInvestmentBeforeTransactionWithCurrencyEffect
		}

		// Compute transaction investment
		transactionInvestment := decimal.Zero
		transactionInvestmentWithCurrencyEffect := decimal.Zero

		if order.Type == "BUY" && order.UnitPriceInBaseCurrency != nil {
			factor := decimal.NewFromInt(int64(GetFactor(order.Type)))
			transactionInvestment = order.Quantity.Mul(*order.UnitPriceInBaseCurrency).Mul(factor)
			transactionInvestmentWithCurrencyEffect = order.Quantity.Mul(*order.UnitPriceInBaseCurrencyWithCurrencyEffect).Mul(factor)

			totalQuantityFromBuyTransactions = totalQuantityFromBuyTransactions.Add(order.Quantity)
			totalInvestmentFromBuyTransactions = totalInvestmentFromBuyTransactions.Add(transactionInvestment)
			totalInvestmentFromBuyTransactionsWithCurrencyEffect = totalInvestmentFromBuyTransactionsWithCurrencyEffect.Add(transactionInvestmentWithCurrencyEffect)
		} else if order.Type == "SELL" && totalUnits.GreaterThan(decimal.Zero) {
			factor := decimal.NewFromInt(int64(GetFactor(order.Type)))
			transactionInvestment = totalInvestment.Div(totalUnits).Mul(order.Quantity).Mul(factor)
			transactionInvestmentWithCurrencyEffect = totalInvestmentWithCurrencyEffect.Div(totalUnits).Mul(order.Quantity).Mul(factor)
		}

		totalInvestmentBeforeTransaction := totalInvestment
		totalInvestmentBeforeTransactionWithCurrencyEffect := totalInvestmentWithCurrencyEffect

		totalInvestment = totalInvestment.Add(transactionInvestment)
		totalInvestmentWithCurrencyEffect = totalInvestmentWithCurrencyEffect.Add(transactionInvestmentWithCurrencyEffect)

		// Capture initial value
		if i >= indexOfStartOrder && initialValue == nil {
			if i == indexOfStartOrder && !valueOfInvestmentBeforeTransaction.IsZero() {
				v := valueOfInvestmentBeforeTransaction
				initialValue = &v
				vfx := valueOfInvestmentBeforeTransactionWithCurrencyEffect
				initialValueWithCurrencyEffect = &vfx
			} else if transactionInvestment.GreaterThan(decimal.Zero) {
				v := transactionInvestment
				initialValue = &v
				vfx := transactionInvestmentWithCurrencyEffect
				initialValueWithCurrencyEffect = &vfx
			}
		}

		// Accumulate fees
		fees = fees.Add(order.FeeInBaseCurrencyCalc)
		feesWithCurrencyEffect = feesWithCurrencyEffect.Add(order.FeeInBaseCurrencyWithCurrencyEffect)

		// Update total units
		totalUnits = totalUnits.Add(order.Quantity.Mul(decimal.NewFromInt(int64(GetFactor(order.Type)))))

		// Value of investment after transaction
		valueOfInvestment := totalUnits.Mul(marketPriceInBaseCurrency)
		valueOfInvestmentWithCurrencyEffect := totalUnits.Mul(marketPriceInBaseCurrencyWithCurrencyEffect)

		// Gross performance from sell
		grossPerformanceFromSell := decimal.Zero
		grossPerformanceFromSellWithCurrencyEffect := decimal.Zero
		if order.Type == "SELL" && order.UnitPriceInBaseCurrency != nil {
			grossPerformanceFromSell = order.UnitPriceInBaseCurrency.Sub(lastAveragePrice).Mul(order.Quantity)
			grossPerformanceFromSellWithCurrencyEffect = order.UnitPriceInBaseCurrencyWithCurrencyEffect.Sub(lastAveragePriceWithCurrencyEffect).Mul(order.Quantity)
		}

		grossPerformanceFromSells = grossPerformanceFromSells.Add(grossPerformanceFromSell)
		grossPerformanceFromSellsWithCurrencyEffect = grossPerformanceFromSellsWithCurrencyEffect.Add(grossPerformanceFromSellWithCurrencyEffect)

		// Update last average price
		if totalQuantityFromBuyTransactions.IsZero() {
			lastAveragePrice = decimal.Zero
			lastAveragePriceWithCurrencyEffect = decimal.Zero
		} else {
			lastAveragePrice = totalInvestmentFromBuyTransactions.Div(totalQuantityFromBuyTransactions)
			lastAveragePriceWithCurrencyEffect = totalInvestmentFromBuyTransactionsWithCurrencyEffect.Div(totalQuantityFromBuyTransactions)
		}

		// Reset on zero quantity
		if totalUnits.IsZero() {
			totalInvestmentFromBuyTransactions = decimal.Zero
			totalInvestmentFromBuyTransactionsWithCurrencyEffect = decimal.Zero
			totalQuantityFromBuyTransactions = decimal.Zero
		}

		// Compute gross performance
		newGrossPerformance := valueOfInvestment.Sub(totalInvestment).Add(grossPerformanceFromSells)
		newGrossPerformanceWithCurrencyEffect := valueOfInvestmentWithCurrencyEffect.Sub(totalInvestmentWithCurrencyEffect).Add(grossPerformanceFromSellsWithCurrencyEffect)
		grossPerformance = newGrossPerformance
		grossPerformanceWithCurrencyEffect = newGrossPerformanceWithCurrencyEffect

		// Capture values at start date
		if order.ItemType == "start" {
			feesAtStartDate = fees
			feesAtStartDateWithCurrencyEffect = feesWithCurrencyEffect
			grossPerformanceAtStartDate = grossPerformance
			grossPerformanceAtStartDateWithCurrencyEffect = grossPerformanceWithCurrencyEffect
		}

		// Record per-date maps (only after start order)
		if i > indexOfStartOrder {
			// Time-weighted investment calculation
			if valueOfInvestmentBeforeTransaction.GreaterThan(decimal.Zero) &&
				(order.Type == "BUY" || order.Type == "SELL") {
				orderDate := ParseDate(order.Date)
				prevDate := ParseDate(orders[i-1].Date)
				daysSinceLast := float64(int(orderDate.Sub(prevDate).Hours() / 24))
				if daysSinceLast <= 0 {
					daysSinceLast = math.SmallestNonzeroFloat64
				}

				totalInvestmentDays += daysSinceLast
				daysDec := decimal.NewFromFloat(daysSinceLast)

				sumOfTimeWeightedInvestments = sumOfTimeWeightedInvestments.Add(
					valueAtStartDate.Sub(*investmentAtStartDate).Add(totalInvestmentBeforeTransaction).Mul(daysDec),
				)
				sumOfTimeWeightedInvestmentsWithCurrencyEffect = sumOfTimeWeightedInvestmentsWithCurrencyEffect.Add(
					valueAtStartDateWithCurrencyEffect.Sub(*investmentAtStartDateWithCurrencyEffect).Add(totalInvestmentBeforeTransactionWithCurrencyEffect).Mul(daysDec),
				)
			}

			currentValues[order.Date] = valueOfInvestment
			currentValuesWithCurrencyEffect[order.Date] = valueOfInvestmentWithCurrencyEffect

			netPerformanceValues[order.Date] = grossPerformance.Sub(grossPerformanceAtStartDate).Sub(fees.Sub(feesAtStartDate))
			netPerformanceValuesWithCurrencyEffect[order.Date] = grossPerformanceWithCurrencyEffect.Sub(grossPerformanceAtStartDateWithCurrencyEffect).Sub(feesWithCurrencyEffect.Sub(feesAtStartDateWithCurrencyEffect))

			investmentValuesAccumulated[order.Date] = totalInvestment
			investmentValuesAccumulatedWithCurrencyEffect[order.Date] = totalInvestmentWithCurrencyEffect

			prev := investmentValuesWithCurrencyEffect[order.Date]
			investmentValuesWithCurrencyEffect[order.Date] = prev.Add(transactionInvestmentWithCurrencyEffect)

			// Time-weighted investment values
			if totalInvestmentDays > math.SmallestNonzeroFloat64 {
				timeWeightedInvestmentValues[order.Date] = sumOfTimeWeightedInvestments.Div(decimal.NewFromFloat(totalInvestmentDays))
				timeWeightedInvestmentValuesWithCurrencyEffect[order.Date] = sumOfTimeWeightedInvestmentsWithCurrencyEffect.Div(decimal.NewFromFloat(totalInvestmentDays))
			} else {
				if totalInvestment.GreaterThan(decimal.Zero) {
					timeWeightedInvestmentValues[order.Date] = totalInvestment
				} else {
					timeWeightedInvestmentValues[order.Date] = decimal.Zero
				}
				if totalInvestmentWithCurrencyEffect.GreaterThan(decimal.Zero) {
					timeWeightedInvestmentValuesWithCurrencyEffect[order.Date] = totalInvestmentWithCurrencyEffect
				} else {
					timeWeightedInvestmentValuesWithCurrencyEffect[order.Date] = decimal.Zero
				}
			}
		}

		if i == indexOfEndOrder {
			break
		}
	}

	// Final scalars
	totalGrossPerformance := grossPerformance.Sub(grossPerformanceAtStartDate)
	totalGrossPerformanceWithCurrencyEffect := grossPerformanceWithCurrencyEffect.Sub(grossPerformanceAtStartDateWithCurrencyEffect)

	totalNetPerformance := grossPerformance.Sub(grossPerformanceAtStartDate).Sub(fees.Sub(feesAtStartDate))

	timeWeightedAvg := decimal.Zero
	timeWeightedAvgFX := decimal.Zero
	if totalInvestmentDays > 0 {
		timeWeightedAvg = sumOfTimeWeightedInvestments.Div(decimal.NewFromFloat(totalInvestmentDays))
		timeWeightedAvgFX = sumOfTimeWeightedInvestmentsWithCurrencyEffect.Div(decimal.NewFromFloat(totalInvestmentDays))
	}

	grossPerformancePercentage := decimal.Zero
	if timeWeightedAvg.GreaterThan(decimal.Zero) {
		grossPerformancePercentage = totalGrossPerformance.Div(timeWeightedAvg)
	}

	grossPerformancePercentageWithCurrencyEffect := decimal.Zero
	if timeWeightedAvgFX.GreaterThan(decimal.Zero) {
		grossPerformancePercentageWithCurrencyEffect = totalGrossPerformanceWithCurrencyEffect.Div(timeWeightedAvgFX)
	}

	netPerformancePercentage := decimal.Zero
	if timeWeightedAvg.GreaterThan(decimal.Zero) {
		netPerformancePercentage = totalNetPerformance.Div(timeWeightedAvg)
	}

	// Per-DateRange maps
	netPerformancePercentageWithCurrencyEffectMap := map[string]decimal.Decimal{}
	netPerformanceWithCurrencyEffectMap := map[string]decimal.Decimal{}

	// Build date ranges: standard + calendar years
	dateRanges := []string{"1d", "1y", "5y", "max", "mtd", "wtd", "ytd"}
	for y := start.Year(); y <= end.Year(); y++ {
		yearStr := fmt.Sprintf("%d", y)
		yearDate := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		now := time.Now().UTC()
		if yearDate.Year() != now.Year() {
			dateRanges = append(dateRanges, yearStr)
		}
	}

	for _, dateRange := range dateRanges {
		rangeStart, rangeEnd := GetIntervalFromDateRange(dateRange)

		if rangeStart.Before(start) {
			rangeStart = start
		}

		rangeEndStr := FormatDate(rangeEnd)
		rangeStartStr := FormatDate(rangeStart)

		currentValAtStart := getOrZero(currentValuesWithCurrencyEffect, rangeStartStr)
		investmentAccumAtStart := getOrZero(investmentValuesAccumulatedWithCurrencyEffect, rangeStartStr)
		grossPerfAtRangeStart := currentValAtStart.Sub(investmentAccumAtStart)

		// Calculate average for this range
		average := decimal.Zero
		dayCount := 0

		for idx := len(c.chartDates) - 1; idx >= 0; idx-- {
			date := c.chartDates[idx]
			if date > rangeEndStr {
				continue
			}
			if date < rangeStartStr {
				break
			}

			if val, ok := investmentValuesAccumulatedWithCurrencyEffect[date]; ok && val.GreaterThan(decimal.Zero) {
				average = average.Add(val.Add(grossPerfAtRangeStart))
				dayCount++
			}
		}

		if dayCount > 0 {
			average = average.Div(decimal.NewFromInt(int64(dayCount)))
		}

		// Net performance for this date range
		npvEnd := getOrZero(netPerformanceValuesWithCurrencyEffect, rangeEndStr)
		var npvStart decimal.Decimal
		if dateRange == "max" {
			npvStart = decimal.Zero
		} else {
			npvStart = getOrZero(netPerformanceValuesWithCurrencyEffect, rangeStartStr)
		}
		netPerformanceWithCurrencyEffectMap[dateRange] = npvEnd.Sub(npvStart)

		if average.GreaterThan(decimal.Zero) {
			netPerformancePercentageWithCurrencyEffectMap[dateRange] = netPerformanceWithCurrencyEffectMap[dateRange].Div(average)
		} else {
			netPerformancePercentageWithCurrencyEffectMap[dateRange] = decimal.Zero
		}
	}

	iv := decimal.Zero
	if initialValue != nil {
		iv = *initialValue
	}
	ivFX := decimal.Zero
	if initialValueWithCurrencyEffect != nil {
		ivFX = *initialValueWithCurrencyEffect
	}

	hasErrors := totalUnits.GreaterThan(decimal.Zero) && (initialValue == nil || unitPriceAtEndDate == nil)

	return SymbolMetrics{
		CurrentValues:                                  currentValues,
		CurrentValuesWithCurrencyEffect:                currentValuesWithCurrencyEffect,
		FeesWithCurrencyEffect:                         feesWithCurrencyEffect,
		GrossPerformance:                               totalGrossPerformance,
		GrossPerformancePercentage:                     grossPerformancePercentage,
		GrossPerformancePercentageWithCurrencyEffect:   grossPerformancePercentageWithCurrencyEffect,
		GrossPerformanceWithCurrencyEffect:             totalGrossPerformanceWithCurrencyEffect,
		HasErrors:                                      hasErrors,
		InitialValue:                                   iv,
		InitialValueWithCurrencyEffect:                 ivFX,
		InvestmentValuesAccumulated:                    investmentValuesAccumulated,
		InvestmentValuesAccumulatedWithCurrencyEffect:  investmentValuesAccumulatedWithCurrencyEffect,
		InvestmentValuesWithCurrencyEffect:             investmentValuesWithCurrencyEffect,
		NetPerformance:                                 totalNetPerformance,
		NetPerformancePercentage:                       netPerformancePercentage,
		NetPerformancePercentageWithCurrencyEffectMap:  netPerformancePercentageWithCurrencyEffectMap,
		NetPerformanceValues:                           netPerformanceValues,
		NetPerformanceValuesWithCurrencyEffect:         netPerformanceValuesWithCurrencyEffect,
		NetPerformanceWithCurrencyEffectMap:            netPerformanceWithCurrencyEffectMap,
		TimeWeightedInvestment:                         timeWeightedAvg,
		TimeWeightedInvestmentValues:                   timeWeightedInvestmentValues,
		TimeWeightedInvestmentValuesWithCurrencyEffect: timeWeightedInvestmentValuesWithCurrencyEffect,
		TimeWeightedInvestmentWithCurrencyEffect:       timeWeightedAvgFX,
		TotalAccountBalanceInBaseCurrency:              decimal.Zero,
		TotalDividend:                                  totalDividend,
		TotalDividendInBaseCurrency:                    totalDividendInBaseCurrency,
		TotalInterest:                                  totalInterest,
		TotalInterestInBaseCurrency:                    totalInterestInBaseCurrency,
		TotalInvestment:                                totalInvestment,
		TotalInvestmentWithCurrencyEffect:              totalInvestmentWithCurrencyEffect,
		TotalLiabilities:                               totalLiabilities,
		TotalLiabilitiesInBaseCurrency:                 totalLiabilitiesInBaseCurrency,
	}
}

// Helper: get market price from nested map.
// Legacy parity: marketSymbolMap is keyed by symbol only in NestJS ROAI.
// Keep this signature/source behavior for strict backward compatibility.
func getMarketPrice(m MarketData, date, symbol string) *decimal.Decimal {
	if dateMap, ok := m[date]; ok {
		if price, ok := dateMap[symbol]; ok {
			return &price
		}
	}
	return nil
}

// Helper: convert float64 to decimal, using fallback if zero
func decFromFloat(v, fallback float64) decimal.Decimal {
	if v == 0 {
		return decimal.NewFromFloat(fallback)
	}
	return decimal.NewFromFloat(v)
}

// Sort key for item types: start=-1, normal=0, end=1
func itemTypeSortKey(itemType string) int {
	switch itemType {
	case "start":
		return -1
	case "end":
		return 1
	default:
		return 0
	}
}
