package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/handler"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
)

func main() {
	var poolCloser func()
	deps := &handler.Deps{}

	cfg, err := db.LoadConfigFromEnv()
	if err != nil {
		log.Printf("database config warning: %v", err)
	} else {
		pool, dbErr := db.Open(context.Background(), cfg)
		if dbErr != nil {
			log.Printf("database connection warning: %v", dbErr)
		} else if pingErr := db.Ping(context.Background(), pool, 5*time.Second); pingErr != nil {
			log.Printf("database ping warning: %v", pingErr)
			pool.Close()
		} else {
			log.Printf("database connected")
			poolCloser = pool.Close
			deps = &handler.Deps{
				AccountService:  service.NewAccountService(pool),
				AccessService:   service.NewAccessService(pool),
				OrderService:    service.NewOrderService(pool),
				PlatformService: service.NewPlatformService(pool),
				TagService:      service.NewTagService(pool),
				UserService:     service.NewUserService(pool),
			}
		}
	}
	if poolCloser != nil {
		defer poolCloser()
	}

	mux := http.NewServeMux()

	// Access
	mux.HandleFunc("GET /api/v1/access", handler.HandleGetAccesses)
	mux.HandleFunc("POST /api/v1/access", handler.HandleCreateAccess)
	mux.HandleFunc("DELETE /api/v1/access/{id}", handler.HandleDeleteAccess)
	mux.HandleFunc("PUT /api/v1/access/{id}", handler.HandleUpdateAccess)

	// Account
	mux.HandleFunc("GET /api/v1/account", handler.HandleGetAccounts)
	mux.HandleFunc("POST /api/v1/account", handler.HandleCreateAccount)
	mux.HandleFunc("POST /api/v1/account/transfer-balance", handler.HandleTransferBalance)
	mux.HandleFunc("DELETE /api/v1/account/{id}", handler.HandleDeleteAccount)
	mux.HandleFunc("GET /api/v1/account/{id}", handler.HandleGetAccount)
	mux.HandleFunc("PUT /api/v1/account/{id}", handler.HandleUpdateAccount)
	mux.HandleFunc("GET /api/v1/account/{id}/balances", handler.HandleGetAccountBalances)

	// Account Balance
	mux.HandleFunc("POST /api/v1/account-balance", handler.HandleCreateAccountBalance)
	mux.HandleFunc("DELETE /api/v1/account-balance/{id}", handler.HandleDeleteAccountBalance)

	// Admin
	mux.HandleFunc("GET /api/v1/admin", handler.HandleGetAdmin)
	mux.HandleFunc("GET /api/v1/admin/demo-user/sync", handler.HandleSyncDemoUser)
	mux.HandleFunc("POST /api/v1/admin/gather", handler.HandleGatherData)
	mux.HandleFunc("POST /api/v1/admin/gather/max", handler.HandleGatherMax)
	mux.HandleFunc("POST /api/v1/admin/gather/profile-data", handler.HandleGatherProfileData)
	mux.HandleFunc("POST /api/v1/admin/gather/profile-data/{dataSource}/{symbol}", handler.HandleGatherProfileDataForSymbol)
	mux.HandleFunc("POST /api/v1/admin/gather/{dataSource}/{symbol}", handler.HandleGatherDataForSymbol)
	mux.HandleFunc("POST /api/v1/admin/gather/{dataSource}/{symbol}/{dateString}", handler.HandleGatherDataForDate)
	mux.HandleFunc("GET /api/v1/admin/market-data", handler.HandleGetAdminMarketData)
	mux.HandleFunc("POST /api/v1/admin/market-data/{dataSource}/{symbol}/test", handler.HandleTestMarketData)
	mux.HandleFunc("DELETE /api/v1/admin/profile-data/{dataSource}/{symbol}", handler.HandleDeleteProfileData)
	mux.HandleFunc("PATCH /api/v1/admin/profile-data/{dataSource}/{symbol}", handler.HandleUpdateProfileData)
	mux.HandleFunc("POST /api/v1/admin/profile-data/{dataSource}/{symbol}", handler.HandleCreateProfileData)
	mux.HandleFunc("PUT /api/v1/admin/settings/{key}", handler.HandleUpdateAdminSetting)
	mux.HandleFunc("GET /api/v1/admin/user", handler.HandleGetAdminUsers)
	mux.HandleFunc("GET /api/v1/admin/user/{id}", handler.HandleGetAdminUser)

	// Admin Queue
	mux.HandleFunc("DELETE /api/v1/admin/queue/job", handler.HandleDeleteQueueJobs)
	mux.HandleFunc("GET /api/v1/admin/queue/job", handler.HandleGetQueueJobs)
	mux.HandleFunc("DELETE /api/v1/admin/queue/job/{id}", handler.HandleDeleteQueueJob)
	mux.HandleFunc("GET /api/v1/admin/queue/job/{id}/execute", handler.HandleExecuteQueueJob)

	// Agent
	mux.HandleFunc("POST /api/v1/agent/chat", handler.HandleCreateAgentChat)
	mux.HandleFunc("POST /api/v1/agent/feedback", handler.HandleCreateAgentFeedback)
	mux.HandleFunc("GET /api/v1/agent/session/{id}", handler.HandleGetAgentSession)

	// AI
	mux.HandleFunc("GET /api/v1/ai/prompt/{mode}", handler.HandleGetAiPrompt)

	// API Keys
	mux.HandleFunc("POST /api/v1/api-keys", handler.HandleCreateApiKey)

	// Asset
	mux.HandleFunc("GET /api/v1/asset/{dataSource}/{symbol}", handler.HandleGetAsset)

	// Assets (web manifest)
	mux.HandleFunc("GET /assets/{languageCode}/site.webmanifest", handler.HandleGetWebManifest)

	// Auth
	mux.HandleFunc("POST /api/v1/auth/anonymous", handler.HandleAuthAnonymous)
	mux.HandleFunc("GET /api/v1/auth/anonymous/{accessToken}", handler.HandleGetAuthAnonymous)
	mux.HandleFunc("GET /api/v1/auth/google", handler.HandleGetAuthGoogle)
	mux.HandleFunc("GET /api/v1/auth/google/callback", handler.HandleGetAuthGoogleCallback)
	mux.HandleFunc("GET /api/v1/auth/oidc", handler.HandleGetAuthOidc)
	mux.HandleFunc("GET /api/v1/auth/oidc/callback", handler.HandleGetAuthOidcCallback)
	mux.HandleFunc("POST /api/v1/auth/webauthn/generate-authentication-options", handler.HandleWebauthnGenerateAuthenticationOptions)
	mux.HandleFunc("GET /api/v1/auth/webauthn/generate-registration-options", handler.HandleGetWebauthnGenerateRegistrationOptions)
	mux.HandleFunc("POST /api/v1/auth/webauthn/verify-attestation", handler.HandleWebauthnVerifyAttestation)
	mux.HandleFunc("POST /api/v1/auth/webauthn/verify-authentication", handler.HandleWebauthnVerifyAuthentication)

	// Auth Device
	mux.HandleFunc("DELETE /api/v1/auth-device/{id}", handler.HandleDeleteAuthDevice)

	// Benchmarks
	mux.HandleFunc("GET /api/v1/benchmarks", handler.HandleGetBenchmarks)
	mux.HandleFunc("POST /api/v1/benchmarks", handler.HandleCreateBenchmark)
	mux.HandleFunc("DELETE /api/v1/benchmarks/{dataSource}/{symbol}", handler.HandleDeleteBenchmark)
	mux.HandleFunc("GET /api/v1/benchmarks/{dataSource}/{symbol}/{startDateString}", handler.HandleGetBenchmarkDetail)

	// Cache
	mux.HandleFunc("POST /api/v1/cache/flush", handler.HandleFlushCache)

	// Data Providers
	mux.HandleFunc("GET /api/v1/data-providers/ghostfolio/asset-profile/{symbol}", handler.HandleGetDataProviderAssetProfile)
	mux.HandleFunc("GET /api/v1/data-providers/ghostfolio/dividends/{symbol}", handler.HandleGetDataProviderDividends)
	mux.HandleFunc("GET /api/v1/data-providers/ghostfolio/historical/{symbol}", handler.HandleGetDataProviderHistorical)
	mux.HandleFunc("GET /api/v1/data-providers/ghostfolio/lookup", handler.HandleDataProviderLookup)
	mux.HandleFunc("GET /api/v1/data-providers/ghostfolio/quotes", handler.HandleGetDataProviderQuotes)
	mux.HandleFunc("GET /api/v1/data-providers/ghostfolio/status", handler.HandleGetDataProviderStatus)

	// Exchange Rate
	mux.HandleFunc("GET /api/v1/exchange-rate/{symbol}/{dateString}", handler.HandleGetExchangeRate)

	// Export
	mux.HandleFunc("GET /api/v1/export", handler.HandleGetExport)

	// Health
	mux.HandleFunc("GET /api/v1/health", handler.HandleGetHealth)
	mux.HandleFunc("GET /api/v1/health/data-enhancer/{name}", handler.HandleGetHealthDataEnhancer)
	mux.HandleFunc("GET /api/v1/health/data-provider/{dataSource}", handler.HandleGetHealthDataProvider)

	// Import
	mux.HandleFunc("POST /api/v1/import", handler.HandleCreateImport)
	mux.HandleFunc("GET /api/v1/import/dividends/{dataSource}/{symbol}", handler.HandleGetImportDividends)

	// Info
	mux.HandleFunc("GET /api/v1/info", handler.HandleGetInfo)

	// Logo
	mux.HandleFunc("GET /api/v1/logo", handler.HandleGetLogo)
	mux.HandleFunc("GET /api/v1/logo/{dataSource}/{symbol}", handler.HandleGetLogoBySymbol)

	// Market Data
	mux.HandleFunc("GET /api/v1/market-data/markets", handler.HandleGetMarkets)
	mux.HandleFunc("GET /api/v1/market-data/{dataSource}/{symbol}", handler.HandleGetMarketData)
	mux.HandleFunc("POST /api/v1/market-data/{dataSource}/{symbol}", handler.HandleCreateMarketData)

	// Order
	mux.HandleFunc("DELETE /api/v1/order", handler.HandleDeleteOrders)
	mux.HandleFunc("GET /api/v1/order", handler.HandleGetOrders)
	mux.HandleFunc("POST /api/v1/order", handler.HandleCreateOrder)
	mux.HandleFunc("DELETE /api/v1/order/{id}", handler.HandleDeleteOrder)
	mux.HandleFunc("GET /api/v1/order/{id}", handler.HandleGetOrder)
	mux.HandleFunc("PUT /api/v1/order/{id}", handler.HandleUpdateOrder)

	// Platform
	mux.HandleFunc("GET /api/v1/platform", handler.HandleGetPlatform)
	mux.HandleFunc("POST /api/v1/platform", handler.HandleCreatePlatform)
	mux.HandleFunc("DELETE /api/v1/platform/{id}", handler.HandleDeletePlatform)
	mux.HandleFunc("PUT /api/v1/platform/{id}", handler.HandleUpdatePlatform)

	// Platforms (read-only)
	mux.HandleFunc("GET /api/v1/platforms", handler.HandleGetPlatforms)

	// Portfolio
	mux.HandleFunc("GET /api/v1/portfolio/details", handler.HandleGetPortfolioDetails)
	mux.HandleFunc("GET /api/v1/portfolio/dividends", handler.HandleGetPortfolioDividends)
	mux.HandleFunc("GET /api/v1/portfolio/holding/{dataSource}/{symbol}", handler.HandleGetPortfolioHolding)
	mux.HandleFunc("PUT /api/v1/portfolio/holding/{dataSource}/{symbol}/tags", handler.HandleUpdatePortfolioHoldingTags)
	mux.HandleFunc("GET /api/v1/portfolio/holdings", handler.HandleGetPortfolioHoldings)
	mux.HandleFunc("GET /api/v1/portfolio/investments", handler.HandleGetPortfolioInvestments)
	mux.HandleFunc("GET /api/v1/portfolio/performance", handler.HandleGetPortfolioPerformance)
	mux.HandleFunc("GET /api/v1/portfolio/report", handler.HandleGetPortfolioReport)

	// Public
	mux.HandleFunc("GET /api/v1/public/{accessId}/portfolio", handler.HandleGetPublicPortfolio)

	// Sitemap
	mux.HandleFunc("GET /sitemap.xml", handler.HandleGetSitemap)

	// Subscription
	mux.HandleFunc("POST /api/v1/subscription/redeem-coupon", handler.HandleRedeemCoupon)
	mux.HandleFunc("GET /api/v1/subscription/stripe/callback", handler.HandleGetStripeCallback)
	mux.HandleFunc("POST /api/v1/subscription/stripe/checkout-session", handler.HandleCreateStripeCheckoutSession)

	// Symbol
	mux.HandleFunc("GET /api/v1/symbol/lookup", handler.HandleSymbolLookup)
	mux.HandleFunc("GET /api/v1/symbol/{dataSource}/{symbol}", handler.HandleGetSymbol)
	mux.HandleFunc("GET /api/v1/symbol/{dataSource}/{symbol}/{dateString}", handler.HandleGetSymbolByDate)

	// Tags
	mux.HandleFunc("GET /api/v1/tags", handler.GetTags(deps))
	mux.HandleFunc("POST /api/v1/tags", handler.CreateTag(deps))
	mux.HandleFunc("DELETE /api/v1/tags/{id}", handler.DeleteTag(deps))
	mux.HandleFunc("PUT /api/v1/tags/{id}", handler.UpdateTag(deps))

	// User
	mux.HandleFunc("DELETE /api/v1/user", handler.HandleDeleteUser)
	mux.HandleFunc("GET /api/v1/user", handler.HandleGetUser)
	mux.HandleFunc("POST /api/v1/user", handler.HandleCreateUser)
	mux.HandleFunc("POST /api/v1/user/access-token", handler.HandleCreateUserAccessToken)
	mux.HandleFunc("PUT /api/v1/user/setting", handler.HandleUpdateUserSetting)
	mux.HandleFunc("DELETE /api/v1/user/{id}", handler.HandleDeleteUserById)
	mux.HandleFunc("POST /api/v1/user/{id}/access-token", handler.HandleCreateUserAccessTokenById)

	// Watchlist
	mux.HandleFunc("GET /api/v1/watchlist", handler.HandleGetWatchlist)
	mux.HandleFunc("POST /api/v1/watchlist", handler.HandleCreateWatchlistItem)
	mux.HandleFunc("DELETE /api/v1/watchlist/{dataSource}/{symbol}", handler.HandleDeleteWatchlistItem)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}

	log.Printf("api-go listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
