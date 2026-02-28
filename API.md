# API Surface Reference (From `apps/api`)

Reference of the original NestJS API to support the Go rewrite in `apps/api-go`.

## Base URL and Versioning

- Default API prefix: `/api/v1` (`setGlobalPrefix('api')` + URI versioning `v1`).
- Prefix exclusions: `/sitemap.xml` and `/<languageCode>/*` (e.g. `/assets/:languageCode/site.webmanifest`).
- Controller routes in source: **116**

## Auth Modes

- `public`: no auth guard
- `jwt`: `AuthGuard('jwt')`
- `api-key`: `AuthGuard('api-key')`
- `oauth-google` / `oauth-oidc`: auth handoff routes

## Notes

- Some endpoints are always mounted but enforce feature flags at runtime (e.g. OIDC and subscription behavior).
- `Headers` only lists explicit controller parameters; required auth headers are represented by `Auth`/`Permission`.

## Go Handler Patterns

All handler code lives in `internal/handler/`. One file per resource (e.g. `order.go`, `account.go`).

Shared helpers are in `internal/handler/handler.go`:

```go
// Response helpers
JSON(w, http.StatusOK, result)                           // Write JSON response
Error(w, http.StatusForbidden, "Forbidden")              // Write {"statusCode":403,"message":"Forbidden"}
NoContent(w)                                             // 204, no body

// Request body
var body CreateOrderRequest
if !Bind(w, r, &body) { return }                         // Decode JSON body; auto-returns 400 on failure

// Query params — common filters (accounts, assetClasses, dataSource, symbol, tags, range, query, holdingType)
filters := ParseFilters(r)

// Query params — pagination (skip, take, sortColumn, sortDirection)
page := ParsePagination(r)

// Query params — one-off
dryRun := r.URL.Query().Get("dryRun") == "true"

// Path params (Go 1.22 built-in)
id := r.PathValue("id")
symbol := r.PathValue("symbol")
dataSource := r.PathValue("dataSource")

// Impersonation header
impersonationId := Impersonation(r)

// Accept-Language header (used by GET /user)
lang := r.Header.Get("Accept-Language")
```

### Handler naming convention

Function names follow `Handle<Verb><Resource>`:

| Pattern         | Example                                     |
| --------------- | ------------------------------------------- |
| GET (list)      | `HandleGetOrders`                           |
| GET (single)    | `HandleGetOrder`                            |
| POST (create)   | `HandleCreateOrder`                         |
| PUT (update)    | `HandleUpdateOrder`                         |
| DELETE (single) | `HandleDeleteOrder`                         |
| DELETE (bulk)   | `HandleDeleteOrders`                        |
| Special         | `HandleTransferBalance`, `HandleFlushCache` |

### Handler template — GET (list, no body)

```go
func HandleGetOrders(w http.ResponseWriter, r *http.Request) {
    filters := ParseFilters(r)
    page := ParsePagination(r)
    _ = filters // TODO: pass to service
    _ = page
    JSON(w, http.StatusOK, map[string]any{"activities": []any{}})
}
```

### Handler template — GET (single, path param)

```go
func HandleGetOrder(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    _ = id // TODO: pass to service
    JSON(w, http.StatusOK, map[string]any{})
}
```

### Handler template — POST/PUT (with body)

```go
func HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
    var body map[string]any
    if !Bind(w, r, &body) { return }
    _ = body // TODO: pass to service
    JSON(w, http.StatusCreated, map[string]any{})
}
```

### Handler template — DELETE (path param)

```go
func HandleDeleteOrder(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    _ = id // TODO: pass to service
    JSON(w, http.StatusOK, map[string]any{})
}
```

### Handler template — composite path params

```go
func HandleGetPortfolioHolding(w http.ResponseWriter, r *http.Request) {
    dataSource := r.PathValue("dataSource")
    symbol := r.PathValue("symbol")
    _ = dataSource
    _ = symbol
    JSON(w, http.StatusOK, map[string]any{})
}
```

### Route registration (in `cmd/server/main.go`)

Routes are registered with Go 1.22 method+pattern syntax. The full route table lives in `main.go` as a single readable block:

```go
mux := http.NewServeMux()

// Health
mux.HandleFunc("GET /api/v1/health", handler.HandleHealth)

// Order
mux.HandleFunc("GET /api/v1/order", handler.HandleGetOrders)
mux.HandleFunc("GET /api/v1/order/{id}", handler.HandleGetOrder)
mux.HandleFunc("POST /api/v1/order", handler.HandleCreateOrder)
mux.HandleFunc("PUT /api/v1/order/{id}", handler.HandleUpdateOrder)
mux.HandleFunc("DELETE /api/v1/order/{id}", handler.HandleDeleteOrder)
mux.HandleFunc("DELETE /api/v1/order", handler.HandleDeleteOrders)
```

Note: Go 1.22 uses `{param}` not `:param` for path parameters.

### Error response format

All errors match the NestJS `HttpException` shape:

```json
{ "statusCode": 403, "message": "Forbidden" }
{ "statusCode": 400, "message": "Invalid request body: ..." }
{ "statusCode": 404, "message": "Not Found" }
```

### Auth (not yet implemented)

Auth middleware will extract the JWT from `Authorization: Bearer <token>` and inject user context. For stubs, all authenticated routes return mock data without validation. Middleware will be added in `internal/middleware/auth.go`.

## Group: `access`

| Method | Path        | Auth | Permission               | Path Params            | Query | Body                  | Headers | Source                      |
| ------ | ----------- | ---- | ------------------------ | ---------------------- | ----- | --------------------- | ------- | --------------------------- |
| GET    | /access     | jwt  | -                        | -                      | -     | -                     | -       | access/access.controller.ts |
| POST   | /access     | jwt  | permissions.createAccess | -                      | -     | data: CreateAccessDto | -       | access/access.controller.ts |
| DELETE | /access/:id | jwt  | permissions.deleteAccess | id: string (arg: 'id') | -     | -                     | -       | access/access.controller.ts |
| PUT    | /access/:id | jwt  | permissions.updateAccess | id: string (arg: 'id') | -     | data: UpdateAccessDto | -       | access/access.controller.ts |

## Group: `account`

| Method | Path                      | Auth | Permission                | Path Params            | Query | Body                   | Headers | Source                        |
| ------ | ------------------------- | ---- | ------------------------- | ---------------------- | ----- | ---------------------- | ------- | ----------------------------- |
| GET    | /account                  | jwt  | -                         | -                      | -     | -                      | -       | account/account.controller.ts |
| POST   | /account                  | jwt  | permissions.createAccount | -                      | -     | data: CreateAccountDto | -       | account/account.controller.ts |
| DELETE | /account/:id              | jwt  | permissions.deleteAccount | id: string (arg: 'id') | -     | -                      | -       | account/account.controller.ts |
| GET    | /account/:id              | jwt  | -                         | id: string (arg: 'id') | -     | -                      | -       | account/account.controller.ts |
| PUT    | /account/:id              | jwt  | permissions.updateAccount | id: string (arg: 'id') | -     | data: UpdateAccountDto | -       | account/account.controller.ts |
| GET    | /account/:id/balances     | jwt  | -                         | id: string (arg: 'id') | -     | -                      | -       | account/account.controller.ts |
| POST   | /account/transfer-balance | jwt  | permissions.updateAccount | -                      | -     | -                      | -       | account/account.controller.ts |

## Group: `account-balance`

| Method | Path                 | Auth | Permission                       | Path Params            | Query | Body                          | Headers | Source                                        |
| ------ | -------------------- | ---- | -------------------------------- | ---------------------- | ----- | ----------------------------- | ------- | --------------------------------------------- |
| POST   | /account-balance     | jwt  | permissions.createAccountBalance | -                      | -     | data: CreateAccountBalanceDto | -       | account-balance/account-balance.controller.ts |
| DELETE | /account-balance/:id | jwt  | permissions.deleteAccountBalance | id: string (arg: 'id') | -     | -                             | -       | account-balance/account-balance.controller.ts |

## Group: `admin`

| Method | Path                                           | Auth | Permission                      | Path Params                                                                                                        | Query                               | Body                                                 | Headers | Source                          |
| ------ | ---------------------------------------------- | ---- | ------------------------------- | ------------------------------------------------------------------------------------------------------------------ | ----------------------------------- | ---------------------------------------------------- | ------- | ------------------------------- |
| GET    | /admin                                         | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| GET    | /admin/demo-user/sync                          | jwt  | permissions.syncDemoUserAccount | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/gather                                  | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/gather/:dataSource/:symbol              | jwt  | permissions.accessAdminControl  | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                         | dateRange: DateRange (arg: 'range') | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/gather/:dataSource/:symbol/:dateString  | jwt  | permissions.accessAdminControl  | dataSource: DataSource (arg: 'dataSource'); dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol') | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/gather/max                              | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/gather/profile-data                     | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/gather/profile-data/:dataSource/:symbol | jwt  | permissions.accessAdminControl  | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                         | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| GET    | /admin/market-data                             | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| POST   | /admin/market-data/:dataSource/:symbol/test    | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | data: { scraperConfiguration: ScraperConfiguration } | -       | admin/admin.controller.ts       |
| DELETE | /admin/profile-data/:dataSource/:symbol        | jwt  | permissions.accessAdminControl  | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                         | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| PATCH  | /admin/profile-data/:dataSource/:symbol        | jwt  | permissions.accessAdminControl  | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                         | -                                   | assetProfile: UpdateAssetProfileDto                  | -       | admin/admin.controller.ts       |
| POST   | /admin/profile-data/:dataSource/:symbol        | jwt  | permissions.accessAdminControl  | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                         | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| DELETE | /admin/queue/job                               | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/queue/queue.controller.ts |
| GET    | /admin/queue/job                               | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/queue/queue.controller.ts |
| DELETE | /admin/queue/job/:id                           | jwt  | permissions.accessAdminControl  | id: string (arg: 'id')                                                                                             | -                                   | -                                                    | -       | admin/queue/queue.controller.ts |
| GET    | /admin/queue/job/:id/execute                   | jwt  | permissions.accessAdminControl  | id: string (arg: 'id')                                                                                             | -                                   | -                                                    | -       | admin/queue/queue.controller.ts |
| PUT    | /admin/settings/:key                           | jwt  | permissions.accessAdminControl  | key: string (arg: 'key')                                                                                           | -                                   | data: UpdatePropertyDto                              | -       | admin/admin.controller.ts       |
| GET    | /admin/user                                    | jwt  | permissions.accessAdminControl  | -                                                                                                                  | -                                   | -                                                    | -       | admin/admin.controller.ts       |
| GET    | /admin/user/:id                                | jwt  | permissions.accessAdminControl  | id: string (arg: 'id')                                                                                             | -                                   | -                                                    | -       | admin/admin.controller.ts       |

## Group: `agent`

| Method | Path               | Auth | Permission                  | Path Params            | Query | Body                       | Headers | Source                    |
| ------ | ------------------ | ---- | --------------------------- | ---------------------- | ----- | -------------------------- | ------- | ------------------------- |
| POST   | /agent/chat        | jwt  | permissions.accessAssistant | -                      | -     | body: AgentChatRequest     | -       | agent/agent.controller.ts |
| POST   | /agent/feedback    | jwt  | permissions.accessAssistant | -                      | -     | body: AgentFeedbackRequest | -       | agent/agent.controller.ts |
| GET    | /agent/session/:id | jwt  | permissions.accessAssistant | id: string (arg: 'id') | -     | -                          | -       | agent/agent.controller.ts |

## Group: `ai`

| Method | Path             | Auth | Permission               | Path Params                      | Query | Body | Headers | Source                        |
| ------ | ---------------- | ---- | ------------------------ | -------------------------------- | ----- | ---- | ------- | ----------------------------- |
| GET    | /ai/prompt/:mode | jwt  | permissions.readAiPrompt | mode: AiPromptMode (arg: 'mode') | -     | -    | -       | endpoints/ai/ai.controller.ts |

## Group: `api-keys`

| Method | Path      | Auth | Permission               | Path Params | Query | Body | Headers | Source                                    |
| ------ | --------- | ---- | ------------------------ | ----------- | ----- | ---- | ------- | ----------------------------------------- |
| POST   | /api-keys | jwt  | permissions.createApiKey | -           | -     | -    | -       | endpoints/api-keys/api-keys.controller.ts |

## Group: `asset`

| Method | Path                       | Auth   | Permission | Path Params                                                                | Query | Body | Headers | Source                    |
| ------ | -------------------------- | ------ | ---------- | -------------------------------------------------------------------------- | ----- | ---- | ------- | ------------------------- |
| GET    | /asset/:dataSource/:symbol | public | -          | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -     | -    | -       | asset/asset.controller.ts |

## Group: `assets`

| Method | Path                                   | Auth   | Permission | Path Params                                | Query | Body | Headers | Source                                |
| ------ | -------------------------------------- | ------ | ---------- | ------------------------------------------ | ----- | ---- | ------- | ------------------------------------- |
| GET    | /assets/:languageCode/site.webmanifest | public | -          | languageCode: string (arg: 'languageCode') | -     | -    | -       | endpoints/assets/assets.controller.ts |

## Group: `auth`

| Method | Path                                           | Auth         | Permission | Path Params                              | Query | Body                                                                | Headers | Source                  |
| ------ | ---------------------------------------------- | ------------ | ---------- | ---------------------------------------- | ----- | ------------------------------------------------------------------- | ------- | ----------------------- |
| POST   | /auth/anonymous                                | public       | -          | -                                        | -     | body: { accessToken: string }                                       | -       | auth/auth.controller.ts |
| GET    | /auth/anonymous/:accessToken                   | public       | -          | accessToken: string (arg: 'accessToken') | -     | -                                                                   | -       | auth/auth.controller.ts |
| GET    | /auth/google                                   | oauth-google | -          | -                                        | -     | -                                                                   | -       | auth/auth.controller.ts |
| GET    | /auth/google/callback                          | oauth-google | -          | -                                        | -     | -                                                                   | -       | auth/auth.controller.ts |
| GET    | /auth/oidc                                     | oauth-oidc   | -          | -                                        | -     | -                                                                   | -       | auth/auth.controller.ts |
| GET    | /auth/oidc/callback                            | oauth-oidc   | -          | -                                        | -     | -                                                                   | -       | auth/auth.controller.ts |
| POST   | /auth/webauthn/generate-authentication-options | public       | -          | -                                        | -     | body: { deviceId: string }                                          | -       | auth/auth.controller.ts |
| GET    | /auth/webauthn/generate-registration-options   | jwt          | -          | -                                        | -     | -                                                                   | -       | auth/auth.controller.ts |
| POST   | /auth/webauthn/verify-attestation              | jwt          | -          | -                                        | -     | body: { deviceName: string; credential: AttestationCredentialJSON } | -       | auth/auth.controller.ts |
| POST   | /auth/webauthn/verify-authentication           | public       | -          | -                                        | -     | body: { deviceId: string; credential: AssertionCredentialJSON }     | -       | auth/auth.controller.ts |

## Group: `auth-device`

| Method | Path             | Auth | Permission                   | Path Params            | Query | Body | Headers | Source                                |
| ------ | ---------------- | ---- | ---------------------------- | ---------------------- | ----- | ---- | ------- | ------------------------------------- |
| DELETE | /auth-device/:id | jwt  | permissions.deleteAuthDevice | id: string (arg: 'id') | -     | -    | -       | auth-device/auth-device.controller.ts |

## Group: `benchmarks`

| Method | Path                                             | Auth   | Permission                     | Path Params                                                                                                                  | Query                                       | Body | Headers | Source                                        |
| ------ | ------------------------------------------------ | ------ | ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- | ---- | ------- | --------------------------------------------- |
| GET    | /benchmarks                                      | public | -                              | -                                                                                                                            | -                                           | -    | -       | endpoints/benchmarks/benchmarks.controller.ts |
| POST   | /benchmarks                                      | jwt    | permissions.accessAdminControl | -                                                                                                                            | -                                           | -    | -       | endpoints/benchmarks/benchmarks.controller.ts |
| DELETE | /benchmarks/:dataSource/:symbol                  | jwt    | permissions.accessAdminControl | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                                   | -                                           | -    | -       | endpoints/benchmarks/benchmarks.controller.ts |
| GET    | /benchmarks/:dataSource/:symbol/:startDateString | jwt    | -                              | dataSource: DataSource (arg: 'dataSource'); startDateString: string (arg: 'startDateString'); symbol: string (arg: 'symbol') | dateRange: DateRange = 'max' (arg: 'range') | -    | -       | endpoints/benchmarks/benchmarks.controller.ts |

## Group: `cache`

| Method | Path         | Auth | Permission                     | Path Params | Query | Body | Headers | Source                    |
| ------ | ------------ | ---- | ------------------------------ | ----------- | ----- | ---- | ------- | ------------------------- |
| POST   | /cache/flush | jwt  | permissions.accessAdminControl | -           | -     | -    | -       | cache/cache.controller.ts |

## Group: `data-providers`

| Method | Path                                             | Auth    | Permission                               | Path Params                    | Query                   | Body | Headers | Source                                                       |
| ------ | ------------------------------------------------ | ------- | ---------------------------------------- | ------------------------------ | ----------------------- | ---- | ------- | ------------------------------------------------------------ |
| GET    | /data-providers/ghostfolio/asset-profile/:symbol | api-key | permissions.enableDataProviderGhostfolio | symbol: string (arg: 'symbol') | -                       | -    | -       | endpoints/data-providers/ghostfolio/ghostfolio.controller.ts |
| GET    | /data-providers/ghostfolio/dividends/:symbol     | api-key | permissions.enableDataProviderGhostfolio | symbol: string (arg: 'symbol') | query: GetDividendsDto  | -    | -       | endpoints/data-providers/ghostfolio/ghostfolio.controller.ts |
| GET    | /data-providers/ghostfolio/historical/:symbol    | api-key | permissions.enableDataProviderGhostfolio | symbol: string (arg: 'symbol') | query: GetHistoricalDto | -    | -       | endpoints/data-providers/ghostfolio/ghostfolio.controller.ts |
| GET    | /data-providers/ghostfolio/lookup                | api-key | permissions.enableDataProviderGhostfolio | -                              | -                       | -    | -       | endpoints/data-providers/ghostfolio/ghostfolio.controller.ts |
| GET    | /data-providers/ghostfolio/quotes                | api-key | permissions.enableDataProviderGhostfolio | -                              | query: GetQuotesDto     | -    | -       | endpoints/data-providers/ghostfolio/ghostfolio.controller.ts |
| GET    | /data-providers/ghostfolio/status                | api-key | permissions.enableDataProviderGhostfolio | -                              | -                       | -    | -       | endpoints/data-providers/ghostfolio/ghostfolio.controller.ts |

## Group: `exchange-rate`

| Method | Path                               | Auth | Permission | Path Params                                                            | Query | Body | Headers | Source                                    |
| ------ | ---------------------------------- | ---- | ---------- | ---------------------------------------------------------------------- | ----- | ---- | ------- | ----------------------------------------- |
| GET    | /exchange-rate/:symbol/:dateString | jwt  | -          | dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol') | -     | -    | -       | exchange-rate/exchange-rate.controller.ts |

## Group: `export`

| Method | Path    | Auth | Permission | Path Params | Query | Body | Headers | Source                      |
| ------ | ------- | ---- | ---------- | ----------- | ----- | ---- | ------- | --------------------------- |
| GET    | /export | jwt  | -          | -           | -     | -    | -       | export/export.controller.ts |

## Group: `health`

| Method | Path                              | Auth   | Permission | Path Params                                | Query | Body | Headers | Source                      |
| ------ | --------------------------------- | ------ | ---------- | ------------------------------------------ | ----- | ---- | ------- | --------------------------- |
| GET    | /health                           | public | -          | -                                          | -     | -    | -       | health/health.controller.ts |
| GET    | /health/data-enhancer/:name       | public | -          | name: string (arg: 'name')                 | -     | -    | -       | health/health.controller.ts |
| GET    | /health/data-provider/:dataSource | public | -          | dataSource: DataSource (arg: 'dataSource') | -     | -    | -       | health/health.controller.ts |

## Group: `import`

| Method | Path                                  | Auth | Permission              | Path Params                                                                | Query | Body                      | Headers | Source                      |
| ------ | ------------------------------------- | ---- | ----------------------- | -------------------------------------------------------------------------- | ----- | ------------------------- | ------- | --------------------------- |
| POST   | /import                               | jwt  | permissions.createOrder | -                                                                          | -     | importData: ImportDataDto | -       | import/import.controller.ts |
| GET    | /import/dividends/:dataSource/:symbol | jwt  | -                       | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -     | -                         | -       | import/import.controller.ts |

## Group: `info`

| Method | Path  | Auth   | Permission | Path Params | Query | Body | Headers | Source                  |
| ------ | ----- | ------ | ---------- | ----------- | ----- | ---- | ------- | ----------------------- |
| GET    | /info | public | -          | -           | -     | -    | -       | info/info.controller.ts |

## Group: `logo`

| Method | Path                      | Auth   | Permission | Path Params                                                                | Query                    | Body | Headers | Source                  |
| ------ | ------------------------- | ------ | ---------- | -------------------------------------------------------------------------- | ------------------------ | ---- | ------- | ----------------------- |
| GET    | /logo                     | public | -          | -                                                                          | url: string (arg: 'url') | -    | -       | logo/logo.controller.ts |
| GET    | /logo/:dataSource/:symbol | public | -          | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -                        | -    | -       | logo/logo.controller.ts |

## Group: `market-data`

| Method | Path                             | Auth | Permission                          | Path Params                                                                | Query | Body                          | Headers | Source                                          |
| ------ | -------------------------------- | ---- | ----------------------------------- | -------------------------------------------------------------------------- | ----- | ----------------------------- | ------- | ----------------------------------------------- |
| GET    | /market-data/:dataSource/:symbol | jwt  | -                                   | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -     | -                             | -       | endpoints/market-data/market-data.controller.ts |
| POST   | /market-data/:dataSource/:symbol | jwt  | -                                   | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -     | data: UpdateBulkMarketDataDto | -       | endpoints/market-data/market-data.controller.ts |
| GET    | /market-data/markets             | jwt  | permissions.readMarketDataOfMarkets | -                                                                          | -     | -                             | -       | endpoints/market-data/market-data.controller.ts |

## Group: `order`

| Method | Path       | Auth | Permission              | Path Params            | Query | Body                 | Headers | Source                    |
| ------ | ---------- | ---- | ----------------------- | ---------------------- | ----- | -------------------- | ------- | ------------------------- |
| DELETE | /order     | jwt  | permissions.deleteOrder | -                      | -     | -                    | -       | order/order.controller.ts |
| GET    | /order     | jwt  | -                       | -                      | -     | -                    | -       | order/order.controller.ts |
| POST   | /order     | jwt  | permissions.createOrder | -                      | -     | data: CreateOrderDto | -       | order/order.controller.ts |
| DELETE | /order/:id | jwt  | permissions.deleteOrder | id: string (arg: 'id') | -     | -                    | -       | order/order.controller.ts |
| GET    | /order/:id | jwt  | -                       | id: string (arg: 'id') | -     | -                    | -       | order/order.controller.ts |
| PUT    | /order/:id | jwt  | permissions.updateOrder | id: string (arg: 'id') | -     | data: UpdateOrderDto | -       | order/order.controller.ts |

## Group: `platform`

| Method | Path          | Auth | Permission                                | Path Params            | Query | Body                    | Headers | Source                          |
| ------ | ------------- | ---- | ----------------------------------------- | ---------------------- | ----- | ----------------------- | ------- | ------------------------------- |
| GET    | /platform     | jwt  | permissions.readPlatformsWithAccountCount | -                      | -     | -                       | -       | platform/platform.controller.ts |
| POST   | /platform     | jwt  | permissions.createPlatform                | -                      | -     | data: CreatePlatformDto | -       | platform/platform.controller.ts |
| DELETE | /platform/:id | jwt  | permissions.deletePlatform                | id: string (arg: 'id') | -     | -                       | -       | platform/platform.controller.ts |
| PUT    | /platform/:id | jwt  | permissions.updatePlatform                | id: string (arg: 'id') | -     | data: UpdatePlatformDto | -       | platform/platform.controller.ts |

## Group: `platforms`

| Method | Path       | Auth | Permission                | Path Params | Query | Body | Headers | Source                                      |
| ------ | ---------- | ---- | ------------------------- | ----------- | ----- | ---- | ------- | ------------------------------------------- |
| GET    | /platforms | jwt  | permissions.readPlatforms | -           | -     | -    | -       | endpoints/platforms/platforms.controller.ts |

## Group: `portfolio`

| Method | Path                                        | Auth | Permission              | Path Params                                                                | Query                                       | Body                       | Headers | Source                            |
| ------ | ------------------------------------------- | ---- | ----------------------- | -------------------------------------------------------------------------- | ------------------------------------------- | -------------------------- | ------- | --------------------------------- |
| GET    | /portfolio/details                          | jwt  | -                       | -                                                                          | dateRange: DateRange = 'max' (arg: 'range') | -                          | -       | portfolio/portfolio.controller.ts |
| GET    | /portfolio/dividends                        | jwt  | -                       | -                                                                          | dateRange: DateRange = 'max' (arg: 'range') | -                          | -       | portfolio/portfolio.controller.ts |
| GET    | /portfolio/holding/:dataSource/:symbol      | jwt  | -                       | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -                                           | -                          | -       | portfolio/portfolio.controller.ts |
| PUT    | /portfolio/holding/:dataSource/:symbol/tags | jwt  | permissions.updateOrder | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -                                           | data: UpdateHoldingTagsDto | -       | portfolio/portfolio.controller.ts |
| GET    | /portfolio/holdings                         | jwt  | -                       | -                                                                          | dateRange: DateRange = 'max' (arg: 'range') | -                          | -       | portfolio/portfolio.controller.ts |
| GET    | /portfolio/investments                      | jwt  | -                       | -                                                                          | dateRange: DateRange = 'max' (arg: 'range') | -                          | -       | portfolio/portfolio.controller.ts |
| GET    | /portfolio/performance                      | jwt  | -                       | -                                                                          | dateRange: DateRange = 'max' (arg: 'range') | -                          | -       | portfolio/portfolio.controller.ts |
| GET    | /portfolio/report                           | jwt  | -                       | -                                                                          | -                                           | -                          | -       | portfolio/portfolio.controller.ts |

## Group: `public`

| Method | Path                        | Auth   | Permission | Path Params                        | Query | Body | Headers | Source                                |
| ------ | --------------------------- | ------ | ---------- | ---------------------------------- | ----- | ---- | ------- | ------------------------------------- |
| GET    | /public/:accessId/portfolio | public | -          | accessId: string (arg: 'accessId') | -     | -    | -       | endpoints/public/public.controller.ts |

## Group: `sitemap.xml`

| Method | Path         | Auth   | Permission | Path Params | Query | Body | Headers | Source                                  |
| ------ | ------------ | ------ | ---------- | ----------- | ----- | ---- | ------- | --------------------------------------- |
| GET    | /sitemap.xml | public | -          | -           | -     | -    | -       | endpoints/sitemap/sitemap.controller.ts |

## Group: `subscription`

| Method | Path                                  | Auth   | Permission | Path Params | Query | Body | Headers | Source                                  |
| ------ | ------------------------------------- | ------ | ---------- | ----------- | ----- | ---- | ------- | --------------------------------------- |
| POST   | /subscription/redeem-coupon           | jwt    | -          | -           | -     | -    | -       | subscription/subscription.controller.ts |
| GET    | /subscription/stripe/callback         | public | -          | -           | -     | -    | -       | subscription/subscription.controller.ts |
| POST   | /subscription/stripe/checkout-session | jwt    | -          | -           | -     | -    | -       | subscription/subscription.controller.ts |

## Group: `symbol`

| Method | Path                                    | Auth   | Permission | Path Params                                                                                                        | Query | Body | Headers | Source                      |
| ------ | --------------------------------------- | ------ | ---------- | ------------------------------------------------------------------------------------------------------------------ | ----- | ---- | ------- | --------------------------- |
| GET    | /symbol/:dataSource/:symbol             | public | -          | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')                                         | -     | -    | -       | symbol/symbol.controller.ts |
| GET    | /symbol/:dataSource/:symbol/:dateString | jwt    | -          | dataSource: DataSource (arg: 'dataSource'); dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol') | -     | -    | -       | symbol/symbol.controller.ts |
| GET    | /symbol/lookup                          | jwt    | -          | -                                                                                                                  | -     | -    | -       | symbol/symbol.controller.ts |

## Group: `tags`

| Method | Path      | Auth | Permission            | Path Params            | Query | Body               | Headers | Source                            |
| ------ | --------- | ---- | --------------------- | ---------------------- | ----- | ------------------ | ------- | --------------------------------- |
| GET    | /tags     | jwt  | permissions.readTags  | -                      | -     | -                  | -       | endpoints/tags/tags.controller.ts |
| POST   | /tags     | jwt  | -                     | -                      | -     | data: CreateTagDto | -       | endpoints/tags/tags.controller.ts |
| DELETE | /tags/:id | jwt  | permissions.deleteTag | id: string (arg: 'id') | -     | -                  | -       | endpoints/tags/tags.controller.ts |
| PUT    | /tags/:id | jwt  | permissions.updateTag | id: string (arg: 'id') | -     | data: UpdateTagDto | -       | endpoints/tags/tags.controller.ts |

## Group: `user`

| Method | Path                   | Auth   | Permission                       | Path Params            | Query | Body                          | Headers                                         | Source                  |
| ------ | ---------------------- | ------ | -------------------------------- | ---------------------- | ----- | ----------------------------- | ----------------------------------------------- | ----------------------- |
| DELETE | /user                  | jwt    | permissions.deleteOwnUser        | -                      | -     | data: DeleteOwnUserDto        | -                                               | user/user.controller.ts |
| GET    | /user                  | jwt    | -                                | -                      | -     | -                             | acceptLanguage: string (arg: 'accept-language') | user/user.controller.ts |
| POST   | /user                  | public | -                                | -                      | -     | -                             | -                                               | user/user.controller.ts |
| DELETE | /user/:id              | jwt    | permissions.deleteUser           | id: string (arg: 'id') | -     | -                             | -                                               | user/user.controller.ts |
| POST   | /user/:id/access-token | jwt    | permissions.accessAdminControl   | id: string (arg: 'id') | -     | -                             | -                                               | user/user.controller.ts |
| POST   | /user/access-token     | jwt    | permissions.updateOwnAccessToken | -                      | -     | data: UpdateOwnAccessTokenDto | -                                               | user/user.controller.ts |
| PUT    | /user/setting          | jwt    | -                                | -                      | -     | data: UpdateUserSettingDto    | -                                               | user/user.controller.ts |

## Group: `watchlist`

| Method | Path                           | Auth | Permission                      | Path Params                                                                | Query | Body                         | Headers | Source                                      |
| ------ | ------------------------------ | ---- | ------------------------------- | -------------------------------------------------------------------------- | ----- | ---------------------------- | ------- | ------------------------------------------- |
| GET    | /watchlist                     | jwt  | permissions.readWatchlist       | -                                                                          | -     | -                            | -       | endpoints/watchlist/watchlist.controller.ts |
| POST   | /watchlist                     | jwt  | permissions.createWatchlistItem | -                                                                          | -     | data: CreateWatchlistItemDto | -       | endpoints/watchlist/watchlist.controller.ts |
| DELETE | /watchlist/:dataSource/:symbol | jwt  | permissions.deleteWatchlistItem | dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol') | -     | -                            | -       | endpoints/watchlist/watchlist.controller.ts |
