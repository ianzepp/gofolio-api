package handler

import (
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
	"github.com/jmoiron/sqlx"
)

// Deps is the dependency container passed to every handler factory function.
// It is constructed once in main() and shared (read-only) across all request
// goroutines — all fields must be safe for concurrent use.
//
// WHY a single struct rather than per-handler injection: The NestJS codebase
// uses Angular-style module-scoped dependency injection, which naturally groups
// all services at the module level. A flat Deps struct approximates that
// structure without introducing a DI framework, keeping handler factories
// simple (func GetOrders(deps *Deps) http.HandlerFunc).
//
// Pool is exposed directly alongside the typed services because some handlers
// need ad-hoc queries or transactions that do not belong in a specific service.
// It may be nil if the server started without a reachable database — callers
// must guard against this.
type Deps struct {
	DB                    *sqlx.DB
	AccountBalanceService *service.AccountBalanceService
	AccountService        *service.AccountService
	AccessService         *service.AccessService
	AdminService          *service.AdminService
	AssetService          *service.AssetService
	BenchmarkService      *service.BenchmarkService
	DataProviderService   *service.DataProviderService
	ExchangeRateService   *service.ExchangeRateService
	ExportService         *service.ExportService
	InfoService           *service.InfoService
	LogoService           *service.LogoService
	MarketDataService     *service.MarketDataService
	OrderService          *service.OrderService
	PlatformService       *service.PlatformService
	SymbolService         *service.SymbolService
	TagService            *service.TagService
	UserService           *service.UserService
	WatchlistService      *service.WatchlistService
}
