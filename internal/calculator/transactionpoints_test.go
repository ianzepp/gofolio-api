package calculator

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTransactionPoints_SingleBuy(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.NewFromFloat(1.5),
			FeeInBaseCurrency: decimal.NewFromFloat(1.5),
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if len(tps) != 1 {
		t.Fatalf("expected 1 transaction point, got %d", len(tps))
	}

	tp := tps[0]
	if tp.Date != "2024-01-15" {
		t.Errorf("date = %q, want %q", tp.Date, "2024-01-15")
	}
	if len(tp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(tp.Items))
	}

	item := tp.Items[0]
	if item.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", item.Symbol)
	}
	if !item.Quantity.Equal(decimal.NewFromInt(10)) {
		t.Errorf("quantity = %s, want 10", item.Quantity)
	}
	if !item.Investment.Equal(decimal.NewFromFloat(1000)) {
		t.Errorf("investment = %s, want 1000", item.Investment)
	}
	if !item.AveragePrice.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("averagePrice = %s, want 100", item.AveragePrice)
	}
	if item.ActivitiesCount != 1 {
		t.Errorf("activitiesCount = %d, want 1", item.ActivitiesCount)
	}
}

func TestTransactionPoints_BuyThenFullSell(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-02-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(120),
			Type:              "SELL",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if len(tps) != 2 {
		t.Fatalf("expected 2 transaction points, got %d", len(tps))
	}

	lastItem := tps[1].Items[0]
	if !lastItem.Quantity.IsZero() {
		t.Errorf("after full sell: quantity = %s, want 0", lastItem.Quantity)
	}
	if !lastItem.Investment.IsZero() {
		t.Errorf("after full sell: investment = %s, want 0", lastItem.Investment)
	}
}

func TestTransactionPoints_BuyAndBuy(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(5),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-02-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(5),
			UnitPrice:         decimal.NewFromFloat(120),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if len(tps) != 2 {
		t.Fatalf("expected 2 transaction points, got %d", len(tps))
	}

	lastItem := tps[1].Items[0]
	if !lastItem.Quantity.Equal(decimal.NewFromInt(10)) {
		t.Errorf("after two buys: quantity = %s, want 10", lastItem.Quantity)
	}
	// Investment = 5*100 + 5*120 = 1100
	expectedInvestment := decimal.NewFromFloat(1100)
	if !lastItem.Investment.Equal(expectedInvestment) {
		t.Errorf("after two buys: investment = %s, want %s", lastItem.Investment, expectedInvestment)
	}
	// AveragePrice = 1100/10 = 110
	if !lastItem.AveragePrice.Equal(decimal.NewFromFloat(110)) {
		t.Errorf("after two buys: averagePrice = %s, want 110", lastItem.AveragePrice)
	}
}

func TestTransactionPoints_PartialSell(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-02-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(3),
			UnitPrice:         decimal.NewFromFloat(120),
			Type:              "SELL",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	lastItem := tps[1].Items[0]
	if !lastItem.Quantity.Equal(decimal.NewFromInt(7)) {
		t.Errorf("after partial sell: quantity = %s, want 7", lastItem.Quantity)
	}
	// Investment = 1000 - 3*100(avgPrice) = 700
	expectedInvestment := decimal.NewFromFloat(700)
	if !lastItem.Investment.Equal(expectedInvestment) {
		t.Errorf("after partial sell: investment = %s, want %s", lastItem.Investment, expectedInvestment)
	}
}

func TestTransactionPoints_MultipleSymbols(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(10),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-01-20",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(5),
			UnitPrice:         decimal.NewFromFloat(200),
			Type:              "BUY",
			Symbol:            "GOOGL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	if len(tps) != 2 {
		t.Fatalf("expected 2 transaction points, got %d", len(tps))
	}

	lastTP := tps[1]
	if len(lastTP.Items) != 2 {
		t.Fatalf("expected 2 items in last tp, got %d", len(lastTP.Items))
	}

	// Items should be sorted by symbol
	if lastTP.Items[0].Symbol != "AAPL" {
		t.Errorf("first item symbol = %q, want AAPL", lastTP.Items[0].Symbol)
	}
	if lastTP.Items[1].Symbol != "GOOGL" {
		t.Errorf("second item symbol = %q, want GOOGL", lastTP.Items[1].Symbol)
	}
}

func TestTransactionPoints_SameDateMerge(t *testing.T) {
	activities := []Activity{
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(5),
			UnitPrice:         decimal.NewFromFloat(100),
			Type:              "BUY",
			Symbol:            "AAPL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
		{
			Date:              "2024-01-15",
			Fee:               decimal.Zero,
			FeeInBaseCurrency: decimal.Zero,
			Quantity:          decimal.NewFromInt(3),
			UnitPrice:         decimal.NewFromFloat(200),
			Type:              "BUY",
			Symbol:            "GOOGL",
			DataSource:        "YAHOO",
			Currency:          "USD",
			AssetSubClass:     "STOCK",
		},
	}

	calc := NewCalculator(activities, nil, "USD")
	tps := calc.GetTransactionPoints()

	// Same-date activities should merge into one transaction point
	if len(tps) != 1 {
		t.Fatalf("expected 1 transaction point (same date merge), got %d", len(tps))
	}

	if len(tps[0].Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(tps[0].Items))
	}
}
