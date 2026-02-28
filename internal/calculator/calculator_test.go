package calculator

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

// assertDecNear checks that two decimals are within epsilon of each other.
func assertDecNear(t *testing.T, field string, got, want decimal.Decimal, epsilon float64) {
	t.Helper()
	diff := got.Sub(want).Abs().InexactFloat64()
	if diff > epsilon {
		t.Errorf("%s = %s, want %s (diff=%f)", field, got, want, diff)
	}
}

func assertFloatNear(t *testing.T, field string, got, want, epsilon float64) {
	t.Helper()
	if math.Abs(got-want) > epsilon {
		t.Errorf("%s = %f, want %f", field, got, want)
	}
}

func TestNoOrders(t *testing.T) {
	calc := NewCalculator(nil, nil, "USD")
	snapshot := calc.ComputeSnapshot(MarketData{}, ExchangeRates{})

	if snapshot.ActivitiesCount != 0 {
		t.Errorf("activitiesCount = %d, want 0", snapshot.ActivitiesCount)
	}
	if snapshot.HasErrors {
		t.Error("hasErrors = true, want false")
	}
	if len(snapshot.Positions) != 0 {
		t.Errorf("positions = %d, want 0", len(snapshot.Positions))
	}
	assertDecNear(t, "currentValue", snapshot.CurrentValueInBaseCurrency, decimal.Zero, 0.01)
}

func TestSingleBuy_SameCurrency(t *testing.T) {
	// Buy 1 share at $100, current price $150, same currency
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.NewFromFloat(1.5),
			FeeInBaseCurrency: decimal.NewFromFloat(1.5),
			Quantity:          decimal.NewFromInt(1),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "TEST",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	today := FormatDate(calc.endDate)

	marketData := MarketData{
		"2024-01-14": {"TEST": decimal.NewFromFloat(99)},
		"2024-01-15": {"TEST": decimal.NewFromFloat(100)},
		today:        {"TEST": decimal.NewFromFloat(150)},
	}
	exchangeRates := ExchangeRates{
		"USDUSD": fillRates("2024-01-01", today, 1.0),
	}

	snapshot := calc.ComputeSnapshot(marketData, exchangeRates)

	if snapshot.HasErrors {
		t.Error("hasErrors = true, want false")
	}
	if snapshot.ActivitiesCount != 1 {
		t.Errorf("activitiesCount = %d, want 1", snapshot.ActivitiesCount)
	}

	// Should have 1 position
	if len(snapshot.Positions) != 1 {
		t.Fatalf("positions = %d, want 1", len(snapshot.Positions))
	}

	pos := snapshot.Positions[0]
	assertDecNear(t, "quantity", pos.Quantity, decimal.NewFromInt(1), 0.001)
	assertDecNear(t, "investment", pos.Investment, decimal.NewFromFloat(100), 1.0)

	// Value should be around 150
	assertDecNear(t, "valueInBaseCurrency", pos.ValueInBaseCurrency, decimal.NewFromFloat(150), 1.0)

	// Gross performance should be around 50
	if pos.GrossPerformance != nil {
		assertDecNear(t, "grossPerformance", *pos.GrossPerformance, decimal.NewFromFloat(50), 1.0)
	}
}

func TestCash(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(1000),
			UnitPrice:         decimal.NewFromInt(1),
			Type:              "BUY",
			Symbol:            "CASH_USD",
			DataSource:        "MANUAL",
			Currency:          "USD",
			AssetSubClass:     "CASH",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	today := FormatDate(calc.endDate)

	exchangeRates := ExchangeRates{
		"USDUSD": fillRates("2024-01-01", today, 1.0),
	}

	snapshot := calc.ComputeSnapshot(MarketData{}, exchangeRates)

	// CASH should not be in positions (includeInHoldings=true but includeInTotalAssetValue=false for CASH)
	// CASH activities are BUY/SELL so includeInHoldings=true, but CASH subclass items are excluded from total asset value
	if snapshot.HasErrors {
		t.Error("hasErrors = true, want false")
	}
}

func TestFeeOnly(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.NewFromFloat(10),
			FeeInBaseCurrency: decimal.NewFromFloat(10),
			Quantity:          decimal.Zero,
			UnitPrice:         decimal.Zero,
			Type:              "FEE",
			Symbol:            "FEE_SYMBOL",
			DataSource:        "MANUAL",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if len(tps) != 1 {
		t.Fatalf("expected 1 tp, got %d", len(tps))
	}

	// FEE should accumulate in fees
	if !tps[0].Fees.Equal(decimal.NewFromFloat(10)) {
		t.Errorf("fees = %s, want 10", tps[0].Fees)
	}

	// Quantity should be 0 (factor=0 for FEE)
	if !tps[0].Items[0].Quantity.IsZero() {
		t.Errorf("quantity = %s, want 0", tps[0].Items[0].Quantity)
	}
}

func TestBuyAndSellWithGain(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "TEST",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-06-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(150),
			Type:              "SELL",
			Symbol:            "TEST",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	today := FormatDate(calc.endDate)

	marketData := MarketData{
		"2024-01-14": {"TEST": decimal.NewFromFloat(99)},
		"2024-01-15": {"TEST": decimal.NewFromFloat(100)},
		"2024-06-15": {"TEST": decimal.NewFromFloat(150)},
		today:        {"TEST": decimal.NewFromFloat(160)},
	}
	exchangeRates := ExchangeRates{
		"USDUSD": fillRates("2024-01-01", today, 1.0),
	}

	snapshot := calc.ComputeSnapshot(marketData, exchangeRates)

	if snapshot.ActivitiesCount != 2 {
		t.Errorf("activitiesCount = %d, want 2", snapshot.ActivitiesCount)
	}
}

func TestBuyWithDividend(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "TEST",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-03-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromFloat(10),
			UnitPrice:         decimal.NewFromFloat(2),
			Type:              "DIVIDEND",
			Symbol:            "TEST",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	today := FormatDate(calc.endDate)

	marketData := MarketData{
		"2024-01-14": {"TEST": decimal.NewFromFloat(99)},
		"2024-01-15": {"TEST": decimal.NewFromFloat(100)},
		"2024-03-15": {"TEST": decimal.NewFromFloat(105)},
		today:        {"TEST": decimal.NewFromFloat(110)},
	}
	exchangeRates := ExchangeRates{
		"USDUSD": fillRates("2024-01-01", today, 1.0),
	}

	snapshot := calc.ComputeSnapshot(marketData, exchangeRates)

	// Should have 1 position with dividend data
	if len(snapshot.Positions) != 1 {
		t.Fatalf("positions = %d, want 1", len(snapshot.Positions))
	}

	pos := snapshot.Positions[0]
	// Dividend = 10 * 2 = 20
	assertDecNear(t, "dividend", pos.Dividend, decimal.NewFromFloat(20), 0.01)
	assertDecNear(t, "dividendInBaseCurrency", pos.DividendInBaseCurrency, decimal.NewFromFloat(20), 0.01)
}

func TestHistoricalData(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "TEST",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	today := FormatDate(calc.endDate)

	marketData := MarketData{
		"2024-01-14": {"TEST": decimal.NewFromFloat(99)},
		"2024-01-15": {"TEST": decimal.NewFromFloat(100)},
		today:        {"TEST": decimal.NewFromFloat(120)},
	}
	exchangeRates := ExchangeRates{
		"USDUSD": fillRates("2024-01-01", today, 1.0),
	}

	snapshot := calc.ComputeSnapshot(marketData, exchangeRates)

	if len(snapshot.HistoricalData) == 0 {
		t.Fatal("expected historical data, got empty")
	}

	// Last entry should have value
	last := snapshot.HistoricalData[len(snapshot.HistoricalData)-1]
	if last.Value == 0 {
		t.Error("last historical data value = 0, expected nonzero")
	}
}

func TestCrossCurrency(t *testing.T) {
	// Buy a EUR stock, portfolio in USD
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.NewFromFloat(5),
			FeeInBaseCurrency: decimal.NewFromFloat(5.5),
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(50),
			Type:              "BUY",
			Symbol:            "SAP",
			DataSource:        "YAHOO",
			Currency:          "EUR",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	today := FormatDate(calc.endDate)

	marketData := MarketData{
		"2024-01-14": {"SAP": decimal.NewFromFloat(49)},
		"2024-01-15": {"SAP": decimal.NewFromFloat(50)},
		today:        {"SAP": decimal.NewFromFloat(60)},
	}
	exchangeRates := ExchangeRates{
		"EURUSD": fillRates("2024-01-01", today, 1.1),
	}

	snapshot := calc.ComputeSnapshot(marketData, exchangeRates)

	if snapshot.HasErrors {
		t.Error("hasErrors = true")
	}
	if len(snapshot.Positions) != 1 {
		t.Fatalf("positions = %d, want 1", len(snapshot.Positions))
	}

	pos := snapshot.Positions[0]
	// Value = 10 * 60 * 1.1 = 660
	assertDecNear(t, "valueInBaseCurrency", pos.ValueInBaseCurrency, decimal.NewFromFloat(660), 1.0)
}

func TestLiability(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromFloat(1),
			UnitPrice:         decimal.NewFromFloat(5000),
			Type:              "LIABILITY",
			Symbol:            "LOAN",
			DataSource:        "MANUAL",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if !tps[0].Liabilities.Equal(decimal.NewFromFloat(5000)) {
		t.Errorf("liabilities = %s, want 5000", tps[0].Liabilities)
	}
}

func TestInterest(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromFloat(1),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "INTEREST",
			Symbol:            "INT",
			DataSource:        "MANUAL",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if !tps[0].Interest.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("interest = %s, want 100", tps[0].Interest)
	}
}

// fillRates creates a map of date->rate for every day in a range.
func fillRates(startStr, endStr string, rate float64) map[string]float64 {
	result := map[string]float64{}
	start := ParseDate(startStr)
	end := ParseDate(endStr)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		result[FormatDate(d)] = rate
	}
	return result
}
