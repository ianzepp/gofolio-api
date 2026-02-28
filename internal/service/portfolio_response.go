package service

import "encoding/json"

// PortfolioFilter holds parsed filter criteria for portfolio queries.
// Empty slices / zero-value strings mean "no filter" (include everything).
type PortfolioFilter struct {
	AccountIDs   []string
	AssetClasses []string
	DataSource   string
	Symbol       string
	Tags         []string
}

// IsEmpty returns true when no filter criteria are set.
func (f *PortfolioFilter) IsEmpty() bool {
	return f == nil || (len(f.AccountIDs) == 0 && len(f.AssetClasses) == 0 &&
		f.DataSource == "" && f.Symbol == "" && len(f.Tags) == 0)
}

// PortfolioHoldingsResponse is the JSON shape for GET /portfolio/holdings.
type PortfolioHoldingsResponse struct {
	Holdings []PortfolioHolding `json:"holdings"`
}

// PortfolioHolding represents a single holding in the portfolio.
type PortfolioHolding struct {
	ActivitiesCount                           int      `json:"activitiesCount"`
	AllocationInPercentage                    float64  `json:"allocationInPercentage"`
	AssetClass                                *string  `json:"assetClass"`
	AssetSubClass                             *string  `json:"assetSubClass"`
	Countries                                 []any    `json:"countries"`
	Currency                                  string   `json:"currency"`
	DataSource                                string   `json:"dataSource"`
	DateOfFirstActivity                       string   `json:"dateOfFirstActivity"`
	Dividend                                  float64  `json:"dividend"`
	GrossPerformance                          *float64 `json:"grossPerformance"`
	GrossPerformancePercent                   *float64 `json:"grossPerformancePercent"`
	GrossPerformancePercentWithCurrencyEffect *float64 `json:"grossPerformancePercentWithCurrencyEffect"`
	GrossPerformanceWithCurrencyEffect        *float64 `json:"grossPerformanceWithCurrencyEffect"`
	Holdings                                  []any    `json:"holdings"`
	Investment                                float64  `json:"investment"`
	MarketPrice                               float64  `json:"marketPrice"`
	MarketState                               string   `json:"marketState,omitempty"`
	Name                                      string   `json:"name"`
	NetPerformance                            *float64 `json:"netPerformance"`
	NetPerformancePercent                     *float64 `json:"netPerformancePercent"`
	NetPerformancePercentWithCurrencyEffect   *float64 `json:"netPerformancePercentWithCurrencyEffect"`
	NetPerformanceWithCurrencyEffect          *float64 `json:"netPerformanceWithCurrencyEffect"`
	Quantity                                  float64  `json:"quantity"`
	Sectors                                   []any    `json:"sectors"`
	Symbol                                    string   `json:"symbol"`
	Tags                                      []any    `json:"tags"`
	URL                                       *string  `json:"url"`
	ValueInBaseCurrency                       *float64 `json:"valueInBaseCurrency"`
	ValueInPercentage                         *float64 `json:"valueInPercentage"`
}

// PortfolioDetailsResponse is the JSON shape for GET /portfolio/details.
type PortfolioDetailsResponse struct {
	Accounts  map[string]AccountDetail    `json:"accounts"`
	HasError  bool                        `json:"hasError"`
	Holdings  map[string]PortfolioHolding `json:"holdings"`
	Platforms map[string]PlatformDetail   `json:"platforms,omitempty"`
	Summary   *PortfolioSummary           `json:"summary,omitempty"`
}

// AccountDetail is the per-account detail in the portfolio details response.
type AccountDetail struct {
	Balance             float64 `json:"balance"`
	Currency            string  `json:"currency"`
	Name                string  `json:"name"`
	ValueInBaseCurrency float64 `json:"valueInBaseCurrency"`
	ValueInPercentage   float64 `json:"valueInPercentage,omitempty"`
}

// PlatformDetail is the per-platform detail in the portfolio details response.
type PlatformDetail struct {
	Balance             float64 `json:"balance"`
	Currency            string  `json:"currency"`
	Name                string  `json:"name"`
	ValueInBaseCurrency float64 `json:"valueInBaseCurrency"`
	ValueInPercentage   float64 `json:"valueInPercentage,omitempty"`
}

// PortfolioSummary contains aggregate portfolio metrics.
type PortfolioSummary struct {
	Cash                       float64 `json:"cash,omitempty"`
	CurrentValueInBaseCurrency float64 `json:"currentValueInBaseCurrency,omitempty"`
	Dividend                   float64 `json:"dividend,omitempty"`
	Fees                       float64 `json:"fees,omitempty"`
	NetWorth                   float64 `json:"netWorth,omitempty"`
	TotalInvestment            float64 `json:"totalInvestment,omitempty"`
}

// PortfolioPerformanceResponse is the JSON shape for GET /portfolio/performance.
type PortfolioPerformanceResponse struct {
	Chart          []map[string]any    `json:"chart"`
	Errors         []map[string]string `json:"errors,omitempty"`
	HasErrors      bool                `json:"hasErrors"`
	FirstOrderDate string              `json:"firstOrderDate"`
	Performance    PerformanceSummary  `json:"performance"`
}

// PerformanceSummary has the aggregate performance numbers.
type PerformanceSummary struct {
	CurrentValueInBaseCurrency                 float64 `json:"currentValueInBaseCurrency"`
	NetPerformance                             float64 `json:"netPerformance"`
	NetPerformancePercentage                   float64 `json:"netPerformancePercentage"`
	NetPerformancePercentageWithCurrencyEffect float64 `json:"netPerformancePercentageWithCurrencyEffect"`
	NetPerformanceWithCurrencyEffect           float64 `json:"netPerformanceWithCurrencyEffect"`
	TotalInvestment                            float64 `json:"totalInvestment"`
	TotalInvestmentValueWithCurrencyEffect     float64 `json:"totalInvestmentValueWithCurrencyEffect"`
}

// PortfolioDividendsResponse is the JSON shape for GET /portfolio/dividends.
type PortfolioDividendsResponse struct {
	Dividends []InvestmentItemResponse `json:"dividends"`
}

// InvestmentItemResponse is a dated investment value.
type InvestmentItemResponse struct {
	Date       string  `json:"date"`
	Investment float64 `json:"investment"`
}

// PortfolioInvestmentsResponse is the JSON shape for GET /portfolio/investments.
type PortfolioInvestmentsResponse struct {
	Investments []InvestmentItemResponse `json:"investments"`
}

// PortfolioReportResponse is the JSON shape for GET /portfolio/report.
type PortfolioReportResponse struct {
	// Stub for now — rules engine is complex and low-priority
}

// PortfolioHoldingDetailResponse is the JSON shape for GET /portfolio/holding/:ds/:sym.
// This matches the legacy NestJS PortfolioHoldingResponse interface exactly.
type PortfolioHoldingDetailResponse struct {
	ActivitiesCount                           int                          `json:"activitiesCount"`
	AveragePrice                              float64                      `json:"averagePrice"`
	DataProviderInfo                          *HoldingDataProviderInfo     `json:"dataProviderInfo"`
	DateOfFirstActivity                       string                       `json:"dateOfFirstActivity"`
	DividendInBaseCurrency                    float64                      `json:"dividendInBaseCurrency"`
	DividendYieldPercent                      float64                      `json:"dividendYieldPercent"`
	DividendYieldPercentWithCurrencyEffect    float64                      `json:"dividendYieldPercentWithCurrencyEffect"`
	FeeInBaseCurrency                         float64                      `json:"feeInBaseCurrency"`
	GrossPerformance                          *float64                     `json:"grossPerformance"`
	GrossPerformancePercent                   *float64                     `json:"grossPerformancePercent"`
	GrossPerformancePercentWithCurrencyEffect *float64                     `json:"grossPerformancePercentWithCurrencyEffect"`
	GrossPerformanceWithCurrencyEffect        *float64                     `json:"grossPerformanceWithCurrencyEffect"`
	HistoricalData                            []map[string]any             `json:"historicalData"`
	InvestmentInBaseCurrencyWithCurrencyEffect float64                     `json:"investmentInBaseCurrencyWithCurrencyEffect"`
	MarketPrice                               float64                      `json:"marketPrice"`
	MarketPriceMax                            float64                      `json:"marketPriceMax"`
	MarketPriceMin                            float64                      `json:"marketPriceMin"`
	NetPerformance                            *float64                     `json:"netPerformance"`
	NetPerformancePercent                     *float64                     `json:"netPerformancePercent"`
	NetPerformancePercentWithCurrencyEffect   *float64                     `json:"netPerformancePercentWithCurrencyEffect"`
	NetPerformanceWithCurrencyEffect          *float64                     `json:"netPerformanceWithCurrencyEffect"`
	Performances                              *HoldingPerformances         `json:"performances"`
	Quantity                                  float64                      `json:"quantity"`
	SymbolProfile                             *HoldingSymbolProfile        `json:"SymbolProfile"`
	Tags                                      []any                        `json:"tags"`
	Value                                     float64                      `json:"value"`
}

// HoldingDataProviderInfo mirrors the legacy DataProviderInfo interface.
type HoldingDataProviderInfo struct {
	DataSource *string `json:"dataSource,omitempty"`
	IsPremium  bool    `json:"isPremium"`
	Name       *string `json:"name,omitempty"`
	URL        *string `json:"url,omitempty"`
}

// HoldingPerformances mirrors the legacy Benchmark['performances'] shape.
type HoldingPerformances struct {
	AllTimeHigh HoldingAllTimeHigh `json:"allTimeHigh"`
}

// HoldingAllTimeHigh is the all-time-high sub-object in performances.
type HoldingAllTimeHigh struct {
	Date             *string  `json:"date"`
	PerformancePercent float64 `json:"performancePercent"`
}

// HoldingSymbolProfile is the nested SymbolProfile in the holding detail response.
// Matches the legacy EnhancedSymbolProfile interface.
type HoldingSymbolProfile struct {
	ActivitiesCount int             `json:"activitiesCount"`
	AssetClass      *string         `json:"assetClass"`
	AssetSubClass   *string         `json:"assetSubClass"`
	Comment         *string         `json:"comment"`
	Countries       json.RawMessage `json:"countries"`
	CreatedAt       string          `json:"createdAt"`
	Currency        *string         `json:"currency"`
	Cusip           *string         `json:"cusip"`
	DataSource      string          `json:"dataSource"`
	Figi            *string         `json:"figi"`
	FigiComposite   *string         `json:"figiComposite"`
	FigiShareClass  *string         `json:"figiShareClass"`
	Holdings        json.RawMessage `json:"holdings"`
	ID              string          `json:"id"`
	IsActive        bool            `json:"isActive"`
	Isin            *string         `json:"isin"`
	Name            *string         `json:"name"`
	Sectors         json.RawMessage `json:"sectors"`
	Symbol          string          `json:"symbol"`
	SymbolMapping   json.RawMessage `json:"symbolMapping"`
	UpdatedAt       string          `json:"updatedAt"`
	URL             *string         `json:"url"`
}
