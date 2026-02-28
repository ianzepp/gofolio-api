package model

// AccessPermission controls the level of access granted.
type AccessPermission string

const (
	AccessPermissionRead           AccessPermission = "READ"
	AccessPermissionReadRestricted AccessPermission = "READ_RESTRICTED"
)

// AssetClass categorizes asset types at the highest level.
type AssetClass string

const (
	AssetClassAlternativeInvestment AssetClass = "ALTERNATIVE_INVESTMENT"
	AssetClassCommodity            AssetClass = "COMMODITY"
	AssetClassEquity               AssetClass = "EQUITY"
	AssetClassFixedIncome          AssetClass = "FIXED_INCOME"
	AssetClassLiquidity            AssetClass = "LIQUIDITY"
	AssetClassRealEstate           AssetClass = "REAL_ESTATE"
)

// AssetSubClass further categorizes assets within an AssetClass.
type AssetSubClass string

const (
	AssetSubClassBond          AssetSubClass = "BOND"
	AssetSubClassCash          AssetSubClass = "CASH"
	AssetSubClassCollectible   AssetSubClass = "COLLECTIBLE"
	AssetSubClassCommodity     AssetSubClass = "COMMODITY"
	AssetSubClassCryptocurrency AssetSubClass = "CRYPTOCURRENCY"
	AssetSubClassETF           AssetSubClass = "ETF"
	AssetSubClassMutualFund    AssetSubClass = "MUTUALFUND"
	AssetSubClassPreciousMetal AssetSubClass = "PRECIOUS_METAL"
	AssetSubClassPrivateEquity AssetSubClass = "PRIVATE_EQUITY"
	AssetSubClassStock         AssetSubClass = "STOCK"
)

// DataSource identifies the provider of market data.
type DataSource string

const (
	DataSourceAlphaVantage          DataSource = "ALPHA_VANTAGE"
	DataSourceCoinGecko             DataSource = "COINGECKO"
	DataSourceEODHistoricalData     DataSource = "EOD_HISTORICAL_DATA"
	DataSourceFinancialModelingPrep DataSource = "FINANCIAL_MODELING_PREP"
	DataSourceGhostfolio            DataSource = "GHOSTFOLIO"
	DataSourceGoogleSheets          DataSource = "GOOGLE_SHEETS"
	DataSourceManual                DataSource = "MANUAL"
	DataSourceRapidAPI              DataSource = "RAPID_API"
	DataSourceYahoo                 DataSource = "YAHOO"
)

// MarketDataState indicates whether market data is final or intraday.
type MarketDataState string

const (
	MarketDataStateClose    MarketDataState = "CLOSE"
	MarketDataStateIntraday MarketDataState = "INTRADAY"
)

// Provider identifies the authentication provider.
type Provider string

const (
	ProviderAnonymous        Provider = "ANONYMOUS"
	ProviderGoogle           Provider = "GOOGLE"
	ProviderInternetIdentity Provider = "INTERNET_IDENTITY"
	ProviderOIDC             Provider = "OIDC"
)

// Role defines user authorization levels.
type Role string

const (
	RoleAdmin    Role = "ADMIN"
	RoleDemo     Role = "DEMO"
	RoleInactive Role = "INACTIVE"
	RoleUser     Role = "USER"
)

// OrderType categorizes the type of an order/activity.
type OrderType string

const (
	OrderTypeBuy       OrderType = "BUY"
	OrderTypeDividend  OrderType = "DIVIDEND"
	OrderTypeFee       OrderType = "FEE"
	OrderTypeInterest  OrderType = "INTEREST"
	OrderTypeLiability OrderType = "LIABILITY"
	OrderTypeSell      OrderType = "SELL"
)

// ViewMode controls how the portfolio is displayed.
type ViewMode string

const (
	ViewModeDefault ViewMode = "DEFAULT"
	ViewModeZen     ViewMode = "ZEN"
)
