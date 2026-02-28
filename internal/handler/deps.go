package handler

import "github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"

// Deps holds service dependencies injected into handlers.
type Deps struct {
	AccountService  *service.AccountService
	AccessService   *service.AccessService
	OrderService    *service.OrderService
	PlatformService *service.PlatformService
	TagService      *service.TagService
	UserService     *service.UserService
}
