// Package model defines the core domain types for the Ghostfolio Go API server.
// These types are a direct mapping of the Prisma/Postgres schema used by the
// original NestJS application, preserving column names via `db` struct tags so
// that sqlx can scan rows without manual aliasing.
//
// Enum types are defined as named string types rather than iota integers so that
// values round-trip through JSON and Postgres without a separate lookup table.
// All string constants match the Prisma enum values verbatim.
package model

// AccessPermission controls the level of access granted to a portfolio viewer.
// READ_RESTRICTED hides sensitive fields (e.g. absolute amounts) from the grantee,
// which is useful for sharing a portfolio publicly without exposing net worth.
type AccessPermission string

const (
	AccessPermissionRead           AccessPermission = "READ"
	AccessPermissionReadRestricted AccessPermission = "READ_RESTRICTED"
)

// AssetClass categorizes financial instruments at the broadest level.
// These values drive top-level portfolio allocation charts and filtering.
// TRADE-OFF: a flat enum is simpler than a taxonomy tree but cannot represent
// mixed-class instruments (e.g. convertible bonds straddle FIXED_INCOME and EQUITY).
type AssetClass string

const (
	AssetClassAlternativeInvestment AssetClass = "ALTERNATIVE_INVESTMENT"
	AssetClassCommodity             AssetClass = "COMMODITY"
	AssetClassEquity                AssetClass = "EQUITY"
	AssetClassFixedIncome           AssetClass = "FIXED_INCOME"
	// WHY: cash and cash-equivalents need their own class so liquidity is
	// tracked separately from invested capital in allocation calculations.
	AssetClassLiquidity  AssetClass = "LIQUIDITY"
	AssetClassRealEstate AssetClass = "REAL_ESTATE"
)

// AssetSubClass further categorizes assets within an AssetClass, enabling
// finer-grained breakdown charts (e.g. ETF vs. STOCK within EQUITY).
// Not all (AssetClass, AssetSubClass) combinations are semantically valid;
// validation is enforced at the NestJS layer rather than at the DB level.
type AssetSubClass string

const (
	AssetSubClassBond           AssetSubClass = "BOND"
	AssetSubClassCash           AssetSubClass = "CASH"
	AssetSubClassCollectible    AssetSubClass = "COLLECTIBLE"
	AssetSubClassCommodity      AssetSubClass = "COMMODITY"
	AssetSubClassCryptocurrency AssetSubClass = "CRYPTOCURRENCY"
	AssetSubClassETF            AssetSubClass = "ETF"
	// NOTE: Prisma schema uses "MUTUALFUND" (no underscore) to match the original
	// data provider values; keep this verbatim to avoid round-trip mismatches.
	AssetSubClassMutualFund    AssetSubClass = "MUTUALFUND"
	AssetSubClassPreciousMetal AssetSubClass = "PRECIOUS_METAL"
	AssetSubClassPrivateEquity AssetSubClass = "PRIVATE_EQUITY"
	AssetSubClassStock         AssetSubClass = "STOCK"
)

// DataSource identifies which data provider supplied price/metadata for a symbol.
// This drives the routing logic in the data provider layer: the system looks up
// the correct provider implementation by matching this enum value. Adding a new
// external provider requires extending this enum and registering a handler.
//
// TRADE-OFF: storing the provider name in the DB couples schema migrations to
// provider lifecycle, but it avoids an extra join to a provider lookup table.
type DataSource string

const (
	DataSourceAlphaVantage          DataSource = "ALPHA_VANTAGE"
	DataSourceCoinGecko             DataSource = "COINGECKO"
	DataSourceEODHistoricalData     DataSource = "EOD_HISTORICAL_DATA"
	DataSourceFinancialModelingPrep DataSource = "FINANCIAL_MODELING_PREP"
	// DataSourceGhostfolio is the Ghostfolio premium data service, distinct from
	// the self-hosted instance — it acts as a curated market data proxy.
	DataSourceGhostfolio DataSource = "GHOSTFOLIO"
	DataSourceGoogleSheets DataSource = "GOOGLE_SHEETS"
	// DataSourceManual indicates user-entered prices with no external sync.
	DataSourceManual   DataSource = "MANUAL"
	DataSourceRapidAPI DataSource = "RAPID_API"
	DataSourceYahoo    DataSource = "YAHOO"
)

// MarketDataState indicates whether a market data point is a confirmed daily
// close or a live intraday snapshot. Portfolio valuation uses CLOSE records for
// historical performance and INTRADAY for the current-day estimate.
// EDGE: on non-trading days only CLOSE records exist; callers must not assume
// an INTRADAY record will always be present for today.
type MarketDataState string

const (
	MarketDataStateClose    MarketDataState = "CLOSE"
	MarketDataStateIntraday MarketDataState = "INTRADAY"
)

// Provider identifies how a user authenticated when their account was created.
// This is set once at registration and controls which auth flow is used on login.
// WHY: storing the provider avoids ambiguity when multiple providers share the
// same email address — the user is always routed back to their original IdP.
type Provider string

const (
	// ProviderAnonymous creates a passwordless guest session using a local token.
	// These accounts are ephemeral and are typically deleted after the session ends.
	ProviderAnonymous        Provider = "ANONYMOUS"
	ProviderGoogle           Provider = "GOOGLE"
	// ProviderInternetIdentity uses the Internet Computer (ICP) blockchain identity.
	ProviderInternetIdentity Provider = "INTERNET_IDENTITY"
	ProviderOIDC             Provider = "OIDC"
)

// Role defines user authorization levels enforced by the API middleware.
// Roles are checked per-request rather than embedded in the JWT, so a role
// change takes effect on the next request without requiring token rotation.
type Role string

const (
	RoleAdmin Role = "ADMIN"
	// RoleDemo is a read-only showcase role; mutations are blocked server-side
	// so that demo accounts can be shared publicly without risk of data change.
	RoleDemo Role = "DEMO"
	// RoleInactive blocks all access, used as a soft-delete or suspension state.
	RoleInactive Role = "INACTIVE"
	RoleUser     Role = "USER"
)

// OrderType categorizes the nature of a portfolio activity (called "Order" in the
// schema but covering all transaction kinds). The type determines how an activity
// affects portfolio calculations: BUY/SELL change holdings, DIVIDEND/INTEREST add
// cash income, FEE reduces returns, and ITEM/LIABILITY/LIQUIDITY handle non-traded
// assets like real estate or personal loans.
type OrderType string

const (
	OrderTypeBuy      OrderType = "BUY"
	OrderTypeDividend OrderType = "DIVIDEND"
	OrderTypeFee      OrderType = "FEE"
	OrderTypeInterest OrderType = "INTEREST"
	// OrderTypeItem tracks a non-traded asset value snapshot (e.g. a car or property).
	OrderTypeItem OrderType = "ITEM"
	// OrderTypeLiability records a debt entry that reduces net worth.
	OrderTypeLiability OrderType = "LIABILITY"
	// OrderTypeLiquidity records a cash movement (deposit/withdrawal) into an account
	// without buying a security.
	OrderTypeLiquidity OrderType = "LIQUIDITY"
	OrderTypeSell      OrderType = "SELL"
)

// ViewMode controls how portfolio values are displayed in the UI.
// WHY: ZEN mode hides absolute monetary amounts and shows only percentages,
// making it safe to use Ghostfolio in public or over-the-shoulder without
// revealing net worth. This is a display preference stored server-side so
// that it persists across devices.
type ViewMode string

const (
	ViewModeDefault ViewMode = "DEFAULT"
	ViewModeZen     ViewMode = "ZEN"
)
