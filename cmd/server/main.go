// Package main is the entrypoint for the api-go server, a Go port of the
// Ghostfolio NestJS API. It wires together infrastructure (Postgres connection
// pool, JWT auth config) and all HTTP route handlers, then starts a
// net/http server on the configured port (default 3334).
//
// Architecture overview:
//
//	┌─────────────────────────────────────────────────────┐
//	│  main()                                             │
//	│   PHASE 1: DB connection (graceful-degraded start)  │
//	│   PHASE 2: Service layer wiring                     │
//	│   PHASE 3: Auth config                              │
//	│   PHASE 4: Route registration                       │
//	│   PHASE 5: HTTP server start                        │
//	└─────────────────────────────────────────────────────┘
//
// Graceful degradation: if DATABASE_URL is missing or the initial ping fails,
// the server still starts. Handlers that receive a nil Pool will return errors
// for database-backed requests while health checks remain available. This
// mirrors the NestJS server's behavior of starting before all providers are
// ready, and keeps the process alive so orchestrators can detect and restart it.
//
// Port selection: defaults to 3334 (one above the NestJS server at 3333) so
// both servers can run side-by-side during the parity testing phase.
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
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/provider"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

// defaultPort runs one above the NestJS server (3333) so both can coexist
// during the parity-testing phase.
const defaultPort = "3334"

func main() {
	var poolCloser func()
	deps := &handler.Deps{}

	// PHASE 1: DATABASE CONNECTION
	// WHY: Non-fatal on startup — the server degrades gracefully so that
	// liveness probes and health endpoints remain available even when the
	// database is temporarily unreachable.
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
			// WHY: Close the pool here — a pool that cannot reach the server
			// wastes OS file descriptors and confuses health checks downstream.
			pool.Close()
		} else {
			log.Printf("database connected")
			dbPool = pool
			poolCloser = pool.Close

			// PHASE 2: SERVICE LAYER WIRING
			// WHY: DataProviderService depends on a concrete provider list, and
			// SymbolService depends on DataProviderService, so those two must be
			// constructed first before the flat Deps struct is filled in.
			yahooProvider := provider.NewYahooFinanceProvider()
			manualProvider := provider.NewManualProvider(pool)
			providers := []provider.DataProvider{yahooProvider, manualProvider}
			dataProviderSvc := service.NewDataProviderService(pool, providers)
			symbolSvc := service.NewSymbolService(pool, dataProviderSvc)

			deps = &handler.Deps{
				Pool:                  pool,
				AccountBalanceService: service.NewAccountBalanceService(pool),
				AccountService:        service.NewAccountService(pool),
				AccessService:         service.NewAccessService(pool),
				AdminService:          service.NewAdminService(pool),
				AssetService:          service.NewAssetService(pool),
				BenchmarkService:      service.NewBenchmarkService(pool),
				DataProviderService:   dataProviderSvc,
				ExchangeRateService:   service.NewExchangeRateService(pool),
				ExportService:         service.NewExportService(pool),
				InfoService:           service.NewInfoService(pool),
				LogoService:           service.NewLogoService(pool),
				MarketDataService:     service.NewMarketDataService(pool),
				OrderService:          service.NewOrderService(pool),
				PlatformService:       service.NewPlatformService(pool),
				SymbolService:         symbolSvc,
				TagService:            service.NewTagService(pool),
				UserService:           service.NewUserService(pool),
				WatchlistService:      service.NewWatchlistService(pool),
			}
		}
	}
	if poolCloser != nil {
		defer poolCloser()
	}

	// PHASE 3: AUTH CONFIG
	// WHY: authCfg holds the JWT secret and pool reference used by every
	// auth middleware closure. Constructing it once here avoids each
	// handler capturing the secret string independently.
	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Printf("warning: JWT_SECRET_KEY not set, auth will reject all requests")
	}
	authCfg := auth.Config{JWTSecret: jwtSecret, Pool: dbPool}

	// PHASE 4: ROUTE REGISTRATION
	// WHY: net/http ServeMux uses method+path patterns (Go 1.22+). Each route
	// is wrapped with either auth.Authenticate (validates JWT, no permission
	// check) or auth.RequirePermission (validates JWT + checks a specific
	// permission string). Unprotected routes are registered directly.
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
	mux.HandleFunc(
		"GET /api/v1/data-providers/ghostfolio/asset-profile/{symbol}",
		auth.RequirePermission(authCfg, auth.EnableDataProviderGhostfolio, handler.GetDataProviderAssetProfile(deps)),
	)

	// Exchange Rate
	mux.HandleFunc("GET /api/v1/exchange-rate/{symbol}/{dateString}", auth.Authenticate(authCfg, handler.GetExchangeRate(deps)))

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
	mux.HandleFunc("GET /api/v1/logo", handler.GetLogo(deps))
	mux.HandleFunc("GET /api/v1/logo/{dataSource}/{symbol}", handler.GetLogoBySymbol(deps))

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
	mux.HandleFunc("GET /api/v1/symbol/lookup", auth.Authenticate(authCfg, handler.SymbolLookup(deps)))
	mux.HandleFunc("GET /api/v1/symbol/{dataSource}/{symbol}", handler.GetSymbol(deps))
	mux.HandleFunc("GET /api/v1/symbol/{dataSource}/{symbol}/{dateString}", auth.Authenticate(authCfg, handler.GetSymbolByDate(deps)))

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
	mux.HandleFunc("GET /api/v1/watchlist", auth.RequirePermission(authCfg, auth.ReadWatchlist, handler.GetWatchlist(deps)))
	mux.HandleFunc("POST /api/v1/watchlist", auth.RequirePermission(authCfg, auth.CreateWatchlistItem, handler.CreateWatchlistItem(deps)))
	mux.HandleFunc("DELETE /api/v1/watchlist/{dataSource}/{symbol}", auth.RequirePermission(authCfg, auth.DeleteWatchlistItem, handler.DeleteWatchlistItem(deps)))

	// PHASE 5: HTTP SERVER START
	// WHY: Explicit timeouts are set on all phases of a request lifecycle to
	// prevent slow-client attacks and runaway queries from exhausting goroutines.
	// ReadHeaderTimeout is intentionally shorter than ReadTimeout to stop
	// header-smuggling probes quickly.
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
