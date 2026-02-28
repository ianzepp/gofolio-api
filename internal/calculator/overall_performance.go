package calculator

import (
	"time"

	"github.com/shopspring/decimal"
)

// calculateOverallPerformance aggregates metrics across all positions.
// Port of ROAI lines 29–124.
func (c *Calculator) calculateOverallPerformance(positions []TimelinePosition) PortfolioSnapshot {
	currentValueInBaseCurrency := decimal.Zero
	grossPerformance := decimal.Zero
	grossPerformanceWithCurrencyEffect := decimal.Zero
	hasErrors := false
	netPerformance := decimal.Zero
	totalFeesWithCurrencyEffect := decimal.Zero
	totalInvestment := decimal.Zero
	totalInvestmentWithCurrencyEffect := decimal.Zero

	for _, pos := range positions {
		if !pos.IncludeInTotalAssetValue {
			continue
		}

		if !pos.FeeInBaseCurrency.IsZero() {
			totalFeesWithCurrencyEffect = totalFeesWithCurrencyEffect.Add(pos.FeeInBaseCurrency)
		}

		if !pos.ValueInBaseCurrency.IsZero() {
			currentValueInBaseCurrency = currentValueInBaseCurrency.Add(pos.ValueInBaseCurrency)
		} else {
			hasErrors = true
		}

		if !pos.Investment.IsZero() {
			totalInvestment = totalInvestment.Add(pos.Investment)
			totalInvestmentWithCurrencyEffect = totalInvestmentWithCurrencyEffect.Add(pos.InvestmentWithCurrencyEffect)
		} else {
			hasErrors = true
		}

		if pos.GrossPerformance != nil {
			grossPerformance = grossPerformance.Add(*pos.GrossPerformance)
			if pos.GrossPerformanceWithCurrencyEffect != nil {
				grossPerformanceWithCurrencyEffect = grossPerformanceWithCurrencyEffect.Add(*pos.GrossPerformanceWithCurrencyEffect)
			}
			if pos.NetPerformance != nil {
				netPerformance = netPerformance.Add(*pos.NetPerformance)
			}
		} else if !pos.Quantity.IsZero() {
			hasErrors = true
		}
	}

	// Count BUY/SELL activities
	activitiesCount := 0
	for _, a := range c.activities {
		if a.Type == "BUY" || a.Type == "SELL" {
			activitiesCount++
		}
	}

	return PortfolioSnapshot{
		ActivitiesCount:                    activitiesCount,
		CreatedAt:                          time.Now().UTC(),
		CurrentValueInBaseCurrency:         currentValueInBaseCurrency,
		Errors:                             []AssetIdentifier{},
		HasErrors:                          hasErrors,
		HistoricalData:                     []HistoricalDataItem{},
		Positions:                          positions,
		TotalFeesWithCurrencyEffect:        totalFeesWithCurrencyEffect,
		TotalInvestment:                    totalInvestment,
		TotalInvestmentWithCurrencyEffect:  totalInvestmentWithCurrencyEffect,
		TotalLiabilitiesWithCurrencyEffect: decimal.Zero,
	}
}
