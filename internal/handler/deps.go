package handler

import "github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"

// Deps holds service dependencies injected into handlers.
type Deps struct {
	AccountService    *service.AccountService
	AccessService     *service.AccessService
	AdminService      *service.AdminService
	AssetService      *service.AssetService
	BenchmarkService  *service.BenchmarkService
	ExportService     *service.ExportService
	InfoService       *service.InfoService
	MarketDataService *service.MarketDataService
	OrderService      *service.OrderService
	PlatformService   *service.PlatformService
	TagService        *service.TagService
	UserService       *service.UserService
}
