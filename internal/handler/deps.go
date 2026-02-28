package handler

import "github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"

// Deps holds service dependencies injected into handlers.
type Deps struct {
	AccountBalanceService *service.AccountBalanceService
	AccountService        *service.AccountService
	AccessService         *service.AccessService
	AdminService          *service.AdminService
	AssetService          *service.AssetService
	BenchmarkService      *service.BenchmarkService
	DataProviderService   *service.DataProviderService
	ExportService         *service.ExportService
	InfoService           *service.InfoService
	MarketDataService     *service.MarketDataService
	OrderService          *service.OrderService
	PlatformService       *service.PlatformService
	SymbolService         *service.SymbolService
	TagService            *service.TagService
	UserService           *service.UserService
}
