package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/db"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/handler"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultPort = "3334"

func main() {
	var poolCloser func()
	deps := &handler.Deps{}

	var dbPool *pgxpool.Pool
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
			dbPool = pool
			poolCloser = pool.Close
			deps = &handler.Deps{
				AccountBalanceService: service.NewAccountBalanceService(pool),
				AccountService:        service.NewAccountService(pool),
				AccessService:         service.NewAccessService(pool),
				AdminService:      service.NewAdminService(pool),
				AssetService:      service.NewAssetService(pool),
				BenchmarkService:  service.NewBenchmarkService(pool),
				ExportService:     service.NewExportService(pool),
				InfoService:       service.NewInfoService(pool),
				MarketDataService: service.NewMarketDataService(pool),
				OrderService:      service.NewOrderService(pool),
				PlatformService:   service.NewPlatformService(pool),
				TagService:        service.NewTagService(pool),
				UserService:       service.NewUserService(pool),
			}
		}
	}
	if poolCloser != nil {
		defer poolCloser()
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Printf("warning: JWT_SECRET_KEY not set, auth will reject all requests")
	}
	authCfg := auth.Config{JWTSecret: jwtSecret, Pool: dbPool}

	mux := http.NewServeMux()

	// Access
	mux.HandleFunc("GET /api/v1/access", auth.Authenticate(authCfg, handler.GetAccesses(deps)))
	mux.HandleFunc("POST /api/v1/access", auth.RequirePermission(authCfg, auth.CreateAccess, handler.CreateAccess(deps)))
	mux.HandleFunc("DELETE /api/v1/access/{id}", auth.RequirePermission(authCfg, auth.DeleteAccess, handler.DeleteAccess(deps)))
	mux.HandleFunc("PUT /api/v1/access/{id}", auth.RequirePermission(authCfg, auth.UpdateAccess, handler.UpdateAccess(deps)))

	// Account
	mux.HandleFunc("GET /api/v1/account", auth.Authenticate(authCfg, handler.GetAccounts(deps)))
	mux.HandleFunc("POST /api/v1/account", auth.RequirePermission(authCfg, auth.CreateAccount, handler.CreateAccount(deps)))
	mux.HandleFunc("POST /api/v1/account/transfer-balance", auth.RequirePermission(authCfg, auth.UpdateAccount, handler.TransferBalance(deps)))
	mux.HandleFunc("DELETE /api/v1/account/{id}", auth.RequirePermission(authCfg, auth.DeleteAccount, handler.DeleteAccount(deps)))
	mux.HandleFunc("GET /api/v1/account/{id}", auth.Authenticate(authCfg, handler.GetAccount(deps)))
	mux.HandleFunc("PUT /api/v1/account/{id}", auth.RequirePermission(authCfg, auth.UpdateAccount, handler.UpdateAccount(deps)))
	mux.HandleFunc("GET /api/v1/account/{id}/balances", auth.Authenticate(authCfg, handler.GetAccountBalances(deps)))

	// Account Balance
	mux.HandleFunc("POST /api/v1/account-balance", auth.RequirePermission(authCfg, auth.CreateAccountBalance, handler.HandleCreateAccountBalance(deps)))
	mux.HandleFunc("DELETE /api/v1/account-balance/{id}", auth.RequirePermission(authCfg, auth.DeleteAccountBalance, handler.HandleDeleteAccountBalance(deps)))

	// Admin
	mux.HandleFunc("GET /api/v1/admin", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.GetAdmin(deps)))
	mux.HandleFunc("GET /api/v1/admin/demo-user/sync", handler.HandleSyncDemoUser)
	mux.HandleFunc("POST /api/v1/admin/gather", handler.HandleGatherData)
	mux.HandleFunc("POST /api/v1/admin/gather/max", handler.HandleGatherMax)
	mux.HandleFunc("POST /api/v1/admin/gather/profile-data", handler.HandleGatherProfileData)
	mux.HandleFunc("POST /api/v1/admin/gather/profile-data/{dataSource}/{symbol}", handler.HandleGatherProfileDataForSymbol)
	mux.HandleFunc("POST /api/v1/admin/gather/{dataSource}/{symbol}", handler.HandleGatherDataForSymbol)
	mux.HandleFunc("POST /api/v1/admin/gather/{dataSource}/{symbol}/{dateString}", handler.HandleGatherDataForDate)
	mux.HandleFunc("GET /api/v1/admin/market-data", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.GetAdminMarketData(deps)))
	mux.HandleFunc("POST /api/v1/admin/market-data/{dataSource}/{symbol}/test", handler.HandleTestMarketData)
	mux.HandleFunc("DELETE /api/v1/admin/profile-data/{dataSource}/{symbol}", handler.HandleDeleteProfileData)
	mux.HandleFunc("PATCH /api/v1/admin/profile-data/{dataSource}/{symbol}", handler.HandleUpdateProfileData)
	mux.HandleFunc("POST /api/v1/admin/profile-data/{dataSource}/{symbol}", handler.HandleCreateProfileData)
	mux.HandleFunc("PUT /api/v1/admin/settings/{key}", handler.HandleUpdateAdminSetting)
	mux.HandleFunc("GET /api/v1/admin/user", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.GetAdminUsers(deps)))
	mux.HandleFunc("GET /api/v1/admin/user/{id}", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.GetAdminUser(deps)))

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
	mux.HandleFunc("GET /api/v1/asset/{dataSource}/{symbol}", handler.GetAsset(deps))

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
	mux.HandleFunc("GET /api/v1/benchmarks", handler.GetBenchmarks(deps))
	mux.HandleFunc("POST /api/v1/benchmarks", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.HandleCreateBenchmark))
	mux.HandleFunc("DELETE /api/v1/benchmarks/{dataSource}/{symbol}", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.HandleDeleteBenchmark))
	mux.HandleFunc("GET /api/v1/benchmarks/{dataSource}/{symbol}/{startDateString}", auth.Authenticate(authCfg, handler.HandleGetBenchmarkDetail))

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
	mux.HandleFunc("GET /api/v1/export", auth.Authenticate(authCfg, handler.GetExport(deps)))

	// Health
	mux.HandleFunc("GET /api/v1/health", handler.HandleGetHealth)
	mux.HandleFunc("GET /api/v1/health/data-enhancer/{name}", handler.HandleGetHealthDataEnhancer)
	mux.HandleFunc("GET /api/v1/health/data-provider/{dataSource}", handler.HandleGetHealthDataProvider)

	// Import
	mux.HandleFunc("POST /api/v1/import", handler.HandleCreateImport)
	mux.HandleFunc("GET /api/v1/import/dividends/{dataSource}/{symbol}", handler.HandleGetImportDividends)

	// Info
	mux.HandleFunc("GET /api/v1/info", handler.GetInfo(deps))

	// Logo
	mux.HandleFunc("GET /api/v1/logo", handler.HandleGetLogo)
	mux.HandleFunc("GET /api/v1/logo/{dataSource}/{symbol}", handler.HandleGetLogoBySymbol)

	// Market Data
	mux.HandleFunc("GET /api/v1/market-data/markets", handler.HandleGetMarkets)
	mux.HandleFunc("GET /api/v1/market-data/{dataSource}/{symbol}", auth.Authenticate(authCfg, handler.GetMarketData(deps)))
	mux.HandleFunc("POST /api/v1/market-data/{dataSource}/{symbol}", handler.HandleCreateMarketData)

	// Order
	mux.HandleFunc("DELETE /api/v1/order", auth.RequirePermission(authCfg, auth.DeleteOrder, handler.DeleteOrders(deps)))
	mux.HandleFunc("GET /api/v1/order", auth.Authenticate(authCfg, handler.GetOrders(deps)))
	mux.HandleFunc("POST /api/v1/order", auth.RequirePermission(authCfg, auth.CreateOrder, handler.CreateOrder(deps)))
	mux.HandleFunc("DELETE /api/v1/order/{id}", auth.RequirePermission(authCfg, auth.DeleteOrder, handler.DeleteOrder(deps)))
	mux.HandleFunc("GET /api/v1/order/{id}", auth.Authenticate(authCfg, handler.GetOrder(deps)))
	mux.HandleFunc("PUT /api/v1/order/{id}", auth.RequirePermission(authCfg, auth.UpdateOrder, handler.UpdateOrder(deps)))

	// Platform
	mux.HandleFunc("GET /api/v1/platform", auth.RequirePermission(authCfg, auth.ReadPlatformsWithAccountCount, handler.GetPlatformsWithAccountCount(deps)))
	mux.HandleFunc("POST /api/v1/platform", auth.RequirePermission(authCfg, auth.CreatePlatform, handler.CreatePlatform(deps)))
	mux.HandleFunc("DELETE /api/v1/platform/{id}", auth.RequirePermission(authCfg, auth.DeletePlatform, handler.DeletePlatform(deps)))
	mux.HandleFunc("PUT /api/v1/platform/{id}", auth.RequirePermission(authCfg, auth.UpdatePlatform, handler.UpdatePlatform(deps)))

	// Platforms (read-only)
	mux.HandleFunc("GET /api/v1/platforms", auth.RequirePermission(authCfg, auth.ReadPlatforms, handler.GetPlatforms(deps)))

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
	mux.HandleFunc("GET /api/v1/symbol/lookup", auth.Authenticate(authCfg, handler.HandleSymbolLookup))
	mux.HandleFunc("GET /api/v1/symbol/{dataSource}/{symbol}", handler.HandleGetSymbol)
	mux.HandleFunc("GET /api/v1/symbol/{dataSource}/{symbol}/{dateString}", auth.Authenticate(authCfg, handler.HandleGetSymbolByDate))

	// Tags
	mux.HandleFunc("GET /api/v1/tags", auth.RequirePermission(authCfg, auth.ReadTags, handler.GetTags(deps)))
	mux.HandleFunc("POST /api/v1/tags", auth.RequireAnyPermission(authCfg, []string{auth.CreateOwnTag, auth.CreateTag}, handler.CreateTag(deps)))
	mux.HandleFunc("DELETE /api/v1/tags/{id}", auth.RequirePermission(authCfg, auth.DeleteTag, handler.DeleteTag(deps)))
	mux.HandleFunc("PUT /api/v1/tags/{id}", auth.RequirePermission(authCfg, auth.UpdateTag, handler.UpdateTag(deps)))

	// User
	mux.HandleFunc("DELETE /api/v1/user", auth.RequirePermission(authCfg, auth.DeleteOwnUser, handler.DeleteUser(deps)))
	mux.HandleFunc("GET /api/v1/user", auth.Authenticate(authCfg, handler.GetUser(deps)))
	mux.HandleFunc("POST /api/v1/user", handler.CreateUser(deps))
	mux.HandleFunc("POST /api/v1/user/access-token", auth.RequirePermission(authCfg, auth.UpdateOwnAccessToken, handler.CreateUserAccessToken(deps)))
	mux.HandleFunc("PUT /api/v1/user/setting", auth.Authenticate(authCfg, handler.UpdateUserSetting(deps)))
	mux.HandleFunc("DELETE /api/v1/user/{id}", auth.RequirePermission(authCfg, auth.DeleteUser, handler.DeleteUserById(deps)))
	mux.HandleFunc("POST /api/v1/user/{id}/access-token", auth.RequirePermission(authCfg, auth.AccessAdminControl, handler.CreateUserAccessTokenById(deps)))

	// Watchlist
	mux.HandleFunc("GET /api/v1/watchlist", auth.RequirePermission(authCfg, auth.ReadWatchlist, handler.HandleGetWatchlist))
	mux.HandleFunc("POST /api/v1/watchlist", auth.RequirePermission(authCfg, auth.CreateWatchlistItem, handler.HandleCreateWatchlistItem))
	mux.HandleFunc("DELETE /api/v1/watchlist/{dataSource}/{symbol}", auth.RequirePermission(authCfg, auth.DeleteWatchlistItem, handler.HandleDeleteWatchlistItem))

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("api-go listening on :%s", port)
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
