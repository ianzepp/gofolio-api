// Package calculator implements the ROAI (Return on Absolute Investment)
// portfolio calculator, ported from the TypeScript implementation in
// apps/api/src/app/portfolio/calculator/roai/portfolio-calculator.ts.
//
// All internal math uses github.com/shopspring/decimal to avoid floating-point
// drift. Conversion to/from float64 happens only at repository read and JSON
// write boundaries.
package calculator

import (
	"time"

	"github.com/shopspring/decimal"
)

// Activity is the calculator's input: one portfolio order/event.
type Activity struct {
	Date              string
	Fee               decimal.Decimal
	FeeInBaseCurrency decimal.Decimal
	Quantity          decimal.Decimal
	UnitPrice         decimal.Decimal
	Type              string // BUY, SELL, DIVIDEND, FEE, INTEREST, ITEM, LIABILITY, LIQUIDITY
	Symbol            string
	DataSource        string
	Currency          string
	AssetClass        string
	AssetSubClass     string
	UserID            string // non-empty for custom asset profiles (skip errors)
	Tags              []Tag
	AccountID         string // nullable in DB; empty string means no account
	Name              string // symbol profile name, for response enrichment
	URL               string // symbol profile URL
	Countries         []any  // symbol profile countries JSON
	Sectors           []any  // symbol profile sectors JSON
}

// Tag represents a user-defined tag attached to an activity.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AccountBalanceItem is a dated account balance observation.
type AccountBalanceItem struct {
	Date  string
	Value decimal.Decimal
}

// TransactionPointSymbol holds cumulative per-symbol state at a transaction point.
type TransactionPointSymbol struct {
	Symbol              string
	DataSource          string
	Currency            string
	AssetSubClass       string
	Quantity            decimal.Decimal
	Investment          decimal.Decimal
	AveragePrice        decimal.Decimal
	Fee                 decimal.Decimal
	FeeInBaseCurrency   decimal.Decimal
	Dividend            decimal.Decimal
	ActivitiesCount     int
	DateOfFirstActivity string
	IncludeInHoldings   bool
	SkipErrors          bool
	Tags                []Tag
}

// TransactionPoint groups all symbol states for a single date.
type TransactionPoint struct {
	Date        string
	Items       []TransactionPointSymbol
	Fees        decimal.Decimal
	Interest    decimal.Decimal
	Liabilities decimal.Decimal
}

// SymbolMetrics holds the per-symbol calculation output.
type SymbolMetrics struct {
	CurrentValues                                  map[string]decimal.Decimal
	CurrentValuesWithCurrencyEffect                map[string]decimal.Decimal
	FeesWithCurrencyEffect                         decimal.Decimal
	GrossPerformance                               decimal.Decimal
	GrossPerformancePercentage                     decimal.Decimal
	GrossPerformancePercentageWithCurrencyEffect   decimal.Decimal
	GrossPerformanceWithCurrencyEffect             decimal.Decimal
	HasErrors                                      bool
	InitialValue                                   decimal.Decimal
	InitialValueWithCurrencyEffect                 decimal.Decimal
	InvestmentValuesAccumulated                    map[string]decimal.Decimal
	InvestmentValuesAccumulatedWithCurrencyEffect  map[string]decimal.Decimal
	InvestmentValuesWithCurrencyEffect             map[string]decimal.Decimal
	NetPerformance                                 decimal.Decimal
	NetPerformancePercentage                       decimal.Decimal
	NetPerformancePercentageWithCurrencyEffectMap  map[string]decimal.Decimal
	NetPerformanceValues                           map[string]decimal.Decimal
	NetPerformanceValuesWithCurrencyEffect         map[string]decimal.Decimal
	NetPerformanceWithCurrencyEffectMap            map[string]decimal.Decimal
	TimeWeightedInvestment                         decimal.Decimal
	TimeWeightedInvestmentValues                   map[string]decimal.Decimal
	TimeWeightedInvestmentValuesWithCurrencyEffect map[string]decimal.Decimal
	TimeWeightedInvestmentWithCurrencyEffect       decimal.Decimal
	TotalAccountBalanceInBaseCurrency              decimal.Decimal
	TotalDividend                                  decimal.Decimal
	TotalDividendInBaseCurrency                    decimal.Decimal
	TotalInterest                                  decimal.Decimal
	TotalInterestInBaseCurrency                    decimal.Decimal
	TotalInvestment                                decimal.Decimal
	TotalInvestmentWithCurrencyEffect              decimal.Decimal
	TotalLiabilities                               decimal.Decimal
	TotalLiabilitiesInBaseCurrency                 decimal.Decimal
}

// TimelinePosition holds per-holding output for one symbol.
type TimelinePosition struct {
	ActivitiesCount                               int                        `json:"activitiesCount"`
	AveragePrice                                  decimal.Decimal            `json:"averagePrice"`
	Currency                                      string                     `json:"currency"`
	DataSource                                    string                     `json:"dataSource"`
	DateOfFirstActivity                           string                     `json:"dateOfFirstActivity"`
	Dividend                                      decimal.Decimal            `json:"dividend"`
	DividendInBaseCurrency                        decimal.Decimal            `json:"dividendInBaseCurrency"`
	Fee                                           decimal.Decimal            `json:"fee"`
	FeeInBaseCurrency                             decimal.Decimal            `json:"feeInBaseCurrency"`
	GrossPerformance                              *decimal.Decimal           `json:"grossPerformance"`
	GrossPerformancePercentage                    *decimal.Decimal           `json:"grossPerformancePercentage"`
	GrossPerformancePercentageWithCurrencyEffect  *decimal.Decimal           `json:"grossPerformancePercentageWithCurrencyEffect"`
	GrossPerformanceWithCurrencyEffect            *decimal.Decimal           `json:"grossPerformanceWithCurrencyEffect"`
	IncludeInHoldings                             bool                       `json:"-"`
	IncludeInTotalAssetValue                      bool                       `json:"-"`
	Investment                                    decimal.Decimal            `json:"investment"`
	InvestmentWithCurrencyEffect                  decimal.Decimal            `json:"investmentWithCurrencyEffect"`
	MarketPrice                                   float64                    `json:"marketPrice"`
	MarketPriceInBaseCurrency                     float64                    `json:"marketPriceInBaseCurrency"`
	NetPerformance                                *decimal.Decimal           `json:"netPerformance"`
	NetPerformancePercentage                      *decimal.Decimal           `json:"netPerformancePercentage"`
	NetPerformancePercentageWithCurrencyEffectMap map[string]decimal.Decimal `json:"netPerformancePercentageWithCurrencyEffectMap"`
	NetPerformanceWithCurrencyEffectMap           map[string]decimal.Decimal `json:"netPerformanceWithCurrencyEffectMap"`
	Quantity                                      decimal.Decimal            `json:"quantity"`
	Symbol                                        string                     `json:"symbol"`
	Tags                                          []Tag                      `json:"tags"`
	TimeWeightedInvestment                        decimal.Decimal            `json:"timeWeightedInvestment"`
	TimeWeightedInvestmentWithCurrencyEffect      decimal.Decimal            `json:"timeWeightedInvestmentWithCurrencyEffect"`
	ValueInBaseCurrency                           decimal.Decimal            `json:"valueInBaseCurrency"`
}

// PortfolioSnapshot is the full calculator output.
type PortfolioSnapshot struct {
	ActivitiesCount                    int                  `json:"activitiesCount"`
	CreatedAt                          time.Time            `json:"createdAt"`
	CurrentValueInBaseCurrency         decimal.Decimal      `json:"currentValueInBaseCurrency"`
	Errors                             []AssetIdentifier    `json:"errors"`
	HasErrors                          bool                 `json:"hasErrors"`
	HistoricalData                     []HistoricalDataItem `json:"historicalData"`
	Positions                          []TimelinePosition   `json:"positions"`
	TotalFeesWithCurrencyEffect        decimal.Decimal      `json:"totalFeesWithCurrencyEffect"`
	TotalInterestWithCurrencyEffect    decimal.Decimal      `json:"totalInterestWithCurrencyEffect"`
	TotalInvestment                    decimal.Decimal      `json:"totalInvestment"`
	TotalInvestmentWithCurrencyEffect  decimal.Decimal      `json:"totalInvestmentWithCurrencyEffect"`
	TotalLiabilitiesWithCurrencyEffect decimal.Decimal      `json:"totalLiabilitiesWithCurrencyEffect"`
}

// HistoricalDataItem is one chart point.
type HistoricalDataItem struct {
	Date                                         string  `json:"date"`
	InvestmentValueWithCurrencyEffect            float64 `json:"investmentValueWithCurrencyEffect"`
	NetPerformance                               float64 `json:"netPerformance"`
	NetPerformanceInPercentage                   float64 `json:"netPerformanceInPercentage"`
	NetPerformanceInPercentageWithCurrencyEffect float64 `json:"netPerformanceInPercentageWithCurrencyEffect"`
	NetPerformanceWithCurrencyEffect             float64 `json:"netPerformanceWithCurrencyEffect"`
	NetWorth                                     float64 `json:"netWorth"`
	TotalAccountBalance                          float64 `json:"totalAccountBalance"`
	TotalInvestment                              float64 `json:"totalInvestment"`
	TotalInvestmentValueWithCurrencyEffect       float64 `json:"totalInvestmentValueWithCurrencyEffect"`
	Value                                        float64 `json:"value"`
	ValueWithCurrencyEffect                      float64 `json:"valueWithCurrencyEffect"`
}

// AssetIdentifier identifies a symbol for error reporting.
type AssetIdentifier struct {
	DataSource string `json:"dataSource"`
	Symbol     string `json:"symbol"`
}

// InvestmentItem is a dated investment value.
type InvestmentItem struct {
	Date       string  `json:"date"`
	Investment float64 `json:"investment"`
}

// MarketData maps date → symbol → price.
type MarketData map[string]map[string]decimal.Decimal

// ExchangeRates maps currency-pair → date → rate.
type ExchangeRates map[string]map[string]float64

// orderItem extends Activity with calculator-internal fields for the symbol
// metrics loop. This is the Go equivalent of PortfolioOrderItem in TypeScript.
type orderItem struct {
	Activity
	ItemType                                  string // "start", "end", or ""
	UnitPriceFromMarketData                   *decimal.Decimal
	UnitPriceInBaseCurrency                   *decimal.Decimal
	UnitPriceInBaseCurrencyWithCurrencyEffect *decimal.Decimal
	FeeInBaseCurrencyCalc                     decimal.Decimal
	FeeInBaseCurrencyWithCurrencyEffect       decimal.Decimal
}
