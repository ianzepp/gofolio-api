# Ghostfolio API (Go)

Drop-in Go replacement for the Ghostfolio NestJS backend. Same REST contract, same database, same Angular frontend.

## Why

A 48-hour public coding challenge: rewrite a production backend from scratch in a different language, with AI assistance, while maintaining full compatibility with the existing frontend and database.

### The original codebase

[Ghostfolio](https://github.com/ghostfolio/ghostfolio) is an open-source portfolio tracker. The existing backend is ~39K LOC of TypeScript on NestJS with Prisma, serving 116+ REST endpoints to an Angular frontend. Under the hood, the hardest code is an 800-line ROAI (Return on Added Investment) portfolio calculator that tracks time-weighted performance across multiple currencies, with dual-tracking for currency effects.

### The thesis

Most of that 39K is framework ceremony — NestJS decorators, module wiring, DI boilerplate. The actual business logic is ~5-8K LOC concentrated in portfolio calculations, order handling, exchange rate conversion, and data provider integrations. The rest is mechanical CRUD.

Go eliminates the ceremony. No decorators, no DI containers, no module system. Functions take requests, return responses. The same backend should land at ~15-18K LOC of Go, with less indirection and faster compilation.

### The approach

1. **Keep the Angular frontend untouched.** It's a compiled SPA that makes HTTP calls — it doesn't care what language the server is. Same REST contract, same JSON shapes, same auth tokens.
2. **Keep the same PostgreSQL database.** Generate Go structs from the existing Prisma schema. Zero migration.
3. **Stubs first, logic second.** Every route gets a stub returning valid empty JSON so the frontend never crashes. Then fill in real implementations starting with the critical path.
4. **Same repo, side by side.** The Go backend lives in `apps/api-go/` next to the original `apps/api/`. One git log tells the whole story.

### Why Go

- LLMs generate correct Go on the first try far more often than Rust (no borrow checker fights at 3am)
- 2-second compile times vs 30-60 seconds for Rust
- The backend is REST + PostgreSQL + Redis + HTTP clients — Go's exact sweet spot
- `net/http` in Go 1.22+ handles routing without any framework
- `shopspring/decimal` is a direct equivalent to the `big.js` used throughout the TypeScript codebase

## Status

**Phase 1 — Stubs** (complete): Every route returns 200 with empty/mock JSON so the Angular frontend loads without errors.

**Phase 2 — DB harness** (in progress): 3-layer architecture (handler → service → repository) with raw `pgx` queries. Tag endpoints are the first vertical slice wired to real DB queries.

## Architecture

```
apps/api-go/
  cmd/server/main.go        # Entry point, wiring, router setup
  internal/
    db/                      # Pool management, DBTX interface, transaction helper
    handler/                 # HTTP handlers (one file per controller)
      deps.go                # Deps struct injected into handlers via closures
    service/                 # Business logic, orchestration
    repository/              # SQL queries, struct scanning
    model/                   # Go structs with db:"column" tags, enums
    middleware/              # Auth (JWT), logging, interceptors
    calculator/              # Portfolio calculator (ROAI port)
    provider/                # Data provider integrations
```

### Layering

```
handler/         → HTTP concerns only (parse request, call service, write response)
  ↓
service/         → Business logic, orchestration, transaction boundaries
  ↓
repository/      → SQL queries, struct scanning, transaction participation
  ↓
db/              → Pool management, DBTX interface, WithTx helper
```

- **Router:** Go 1.22+ stdlib (`net/http` with method+pattern routing)
- **Database:** `pgx/v5` with `pgxpool` (raw SQL, no ORM)
- **Scanning:** `pgx.RowToStructByName` with `db:"columnName"` struct tags
- **Cache:** `go-redis` (planned)
- **Decimal math:** `shopspring/decimal` (planned, deferred to calculator layer)
- **No frameworks.** Standard library + minimal dependencies.

### Dependency injection

Handlers receive a `*handler.Deps` struct via closures instead of a DI container:

```go
// main.go — wire services into deps
deps := &handler.Deps{
    TagService:      service.NewTagService(pool),
    AccountService:  service.NewAccountService(pool),
    // ...
}

// Route registration — closure captures deps
mux.HandleFunc("GET /api/v1/tags", handler.GetTags(deps))

// handler/tag.go — handler is a closure factory
func GetTags(deps *Deps) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        tags, err := deps.TagService.GetTags(r.Context())
        // ...
    }
}
```

Stub handlers that aren't yet wired to services remain as plain `func(w, r)` functions. They will be converted to the closure pattern incrementally.

## Database

### Connection pool

The Go server initializes a PostgreSQL connection pool at startup using
`pgxpool` and the existing `DATABASE_URL` value.

- Reads `DATABASE_URL` from the environment (required)
- Creates a shared `*pgxpool.Pool` at startup
- Performs a startup ping to fail early on invalid DB connectivity
- Closes the pool on process shutdown

Optional pool tuning env vars:

- `DB_CONNECT_TIMEOUT` (duration, default `10s`)
- `DB_MAX_CONNS` (default `10`)
- `DB_MIN_CONNS` (default `1`)
- `DB_MAX_CONN_LIFETIME` (duration, default `1h`)
- `DB_MAX_CONN_IDLE_TIME` (duration, default `15m`)
- `DB_HEALTH_CHECK_PERIOD` (duration, default `1m`)

### DBTX interface

The `db.DBTX` interface abstracts over `*pgxpool.Pool` and `pgx.Tx`, so repositories work identically inside or outside a transaction:

```go
type DBTX interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

### Transaction patterns

Mapped from the NestJS/Prisma patterns:

| NestJS Pattern                               | Go Equivalent                                             |
| -------------------------------------------- | --------------------------------------------------------- |
| `prisma.$transaction(async (tx) => { ... })` | `db.WithTx(ctx, pool, func(tx pgx.Tx) error { ... })`     |
| `prisma.$transaction([p1, p2])`              | Same `WithTx` — sequential repo calls inside the callback |
| No explicit transaction (single query)       | Repository called with pool directly (auto-commit)        |

Services hold the `*pgxpool.Pool` and create repositories per call. For transactions, they use `db.WithTx` and pass the `pgx.Tx` to repository constructors:

```go
// Non-transactional — repo uses pool (auto-commit)
func (s *TagService) GetTags(ctx context.Context) ([]model.Tag, error) {
    repo := repository.NewTagRepository(s.pool)
    return repo.FindAll(ctx)
}

// Transactional — repo uses tx
func (s *AccountService) TransferBalance(ctx context.Context, req TransferRequest) error {
    return db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
        repo := repository.NewAccountRepository(tx)
        if err := repo.UpdateBalance(ctx, req.FromID, req.FromUserID, -req.Amount); err != nil {
            return err
        }
        return repo.UpdateBalance(ctx, req.ToID, req.ToUserID, req.Amount)
    })
}
```

### Models

Go structs in `internal/model/` map 1:1 to the 14 Prisma models. Conventions:

- `db:"columnName"` tags match PostgreSQL column names (camelCase, as Prisma preserves them)
- Nullable DB columns → Go pointer types (`*string`, `*time.Time`)
- `Float` in Prisma → `float64` in Go (`shopspring/decimal` deferred to calculator layer)
- JSON columns → `json.RawMessage` (defer parsing to service layer)
- Enums → Go `string` types with constants (e.g., `model.Role`, `model.OrderType`)
- M:M relations (Order↔Tag) → not embedded; loaded via separate queries

### Repositories

Each repository is constructed with a `db.DBTX` and provides typed query methods:

```go
repo := repository.NewTagRepository(pool) // or tx
tags, err := repo.FindAll(ctx)
tag, err := repo.FindByID(ctx, id)
tag, err := repo.Create(ctx, tag)
```

Scanning uses `pgx.CollectRows(rows, pgx.RowToStructByName[T])` — no manual field scanning.

### Differences vs legacy NestJS API

- Legacy uses Prisma (`PrismaClient`) with its own connection/pooling behavior
- Legacy attempts to connect during module init and logs errors, but does not hard-fail process startup
- Go server fail-fasts on missing `DATABASE_URL`, connection errors, or ping failures
- Go uses raw SQL with `pgx` instead of Prisma's query builder — more explicit, no codegen

## Running

```bash
cd apps/api-go
export DATABASE_URL="postgresql://user:password@localhost:5432/ghostfolio?sslmode=prefer"
go run ./cmd/server
# Listens on :3333 — same port as the NestJS backend
```

Startup now opens a PostgreSQL connection pool via `pgxpool` and fails fast if DB connect/ping fails.

## Testing

Current test pattern:

- **Unit/handler tests** in `internal/handler/*_test.go`
  - Use Go's built-in `testing` package
  - Use `httptest` for request/response assertions
  - Prefer table-driven test cases for route behavior
- **Differential integration tests** in `tests/integration`
  - Run the same request fixtures against legacy NestJS and the Go API
  - Compare status (and JSON where configured) to detect parity regressions
  - Details: [`tests/integration/README.md`](./tests/integration/README.md)

Run all tests:

```bash
cd apps/api-go
go test ./...
```

Run with race detector:

```bash
cd apps/api-go
go test -race ./...
```

Run differential integration tests:

```bash
cd apps/api-go
LEGACY_BASE_URL=http://localhost:3333 \
GO_BASE_URL=http://localhost:3334 \
go test ./tests/integration -v
```

Optional auth env vars for integration fixtures:

- `JWT_TOKEN`
- `API_KEY`

## Route Surface

All routes are prefixed with `/api/v1`.

### Auth patterns

- **JWT** — Most endpoints. `Authorization: Bearer <token>` header.
- **API Key** — Data provider endpoints. `Authorization: Bearer <api-key>` header.
- **Public** — No auth required (health, info, logos, public portfolio, etc.).
- **Impersonation** — Optional `x-ghostfolio-impersonation` header on authenticated routes.

### Phase 1 — App boots and navigates (15 endpoints)

These are required for the Angular frontend to load and render the main views.

| Method | Path                     | Purpose                         |
| ------ | ------------------------ | ------------------------------- |
| GET    | `/info`                  | System info (bootstraps the UI) |
| GET    | `/user`                  | Current user                    |
| POST   | `/auth/anonymous`        | Anonymous/token login           |
| GET    | `/portfolio/details`     | Full portfolio view             |
| GET    | `/portfolio/holdings`    | Holdings list                   |
| GET    | `/portfolio/performance` | Performance chart (v2)          |
| GET    | `/order`                 | Activities list                 |
| GET    | `/account`               | Accounts list                   |
| GET    | `/health`                | Health check                    |
| GET    | `/benchmarks`            | Benchmark list                  |
| GET    | `/tags`                  | Tags list                       |
| GET    | `/platforms`             | Platforms list (read-only)      |
| GET    | `/watchlist`             | Watchlist items                 |
| GET    | `/symbol/lookup`         | Symbol search                   |
| PUT    | `/user/setting`          | Update user settings            |

### Phase 2 — CRUD works (26 endpoints)

| Method | Path                             | Purpose                       |
| ------ | -------------------------------- | ----------------------------- |
| POST   | `/order`                         | Create activity               |
| PUT    | `/order/:id`                     | Update activity               |
| DELETE | `/order/:id`                     | Delete activity               |
| DELETE | `/order`                         | Bulk delete activities        |
| GET    | `/order/:id`                     | Get single activity           |
| POST   | `/account`                       | Create account                |
| PUT    | `/account/:id`                   | Update account                |
| DELETE | `/account/:id`                   | Delete account                |
| GET    | `/account/:id`                   | Get single account            |
| GET    | `/account/:id/balances`          | Get account balances          |
| POST   | `/account/transfer-balance`      | Transfer balance              |
| POST   | `/account-balance`               | Create account balance        |
| DELETE | `/account-balance/:id`           | Delete account balance        |
| POST   | `/tags`                          | Create tag                    |
| PUT    | `/tags/:id`                      | Update tag                    |
| DELETE | `/tags/:id`                      | Delete tag                    |
| POST   | `/platform`                      | Create platform               |
| GET    | `/platform`                      | Platforms with account counts |
| PUT    | `/platform/:id`                  | Update platform               |
| DELETE | `/platform/:id`                  | Delete platform               |
| POST   | `/watchlist`                     | Add watchlist item            |
| DELETE | `/watchlist/:dataSource/:symbol` | Remove watchlist item         |
| GET    | `/access`                        | Get access grants             |
| POST   | `/access`                        | Create access grant           |
| PUT    | `/access/:id`                    | Update access grant           |
| DELETE | `/access/:id`                    | Delete access grant           |

### Phase 3 — Full feature parity (remaining ~75 endpoints)

#### Portfolio (remaining)

| Method | Path                                          | Purpose                |
| ------ | --------------------------------------------- | ---------------------- |
| GET    | `/portfolio/dividends`                        | Dividend history       |
| GET    | `/portfolio/holding/:dataSource/:symbol`      | Single holding detail  |
| GET    | `/portfolio/investments`                      | Investment history     |
| GET    | `/portfolio/report`                           | Portfolio X-Ray report |
| PUT    | `/portfolio/holding/:dataSource/:symbol/tags` | Update holding tags    |

#### Auth

| Method | Path                                             | Purpose                        |
| ------ | ------------------------------------------------ | ------------------------------ |
| GET    | `/auth/anonymous/:accessToken`                   | Anonymous login (deprecated)   |
| GET    | `/auth/google`                                   | Google OAuth2 initiate         |
| GET    | `/auth/google/callback`                          | Google OAuth2 callback         |
| GET    | `/auth/oidc`                                     | OIDC login initiate            |
| GET    | `/auth/oidc/callback`                            | OIDC callback                  |
| POST   | `/auth/webauthn/generate-authentication-options` | WebAuthn auth options          |
| GET    | `/auth/webauthn/generate-registration-options`   | WebAuthn registration options  |
| POST   | `/auth/webauthn/verify-attestation`              | WebAuthn verify attestation    |
| POST   | `/auth/webauthn/verify-authentication`           | WebAuthn verify authentication |

#### User (remaining)

| Method | Path                     | Purpose                          |
| ------ | ------------------------ | -------------------------------- |
| POST   | `/user`                  | Create new user (signup)         |
| POST   | `/user/access-token`     | Rotate own access token          |
| POST   | `/user/:id/access-token` | Rotate user access token (admin) |
| DELETE | `/user`                  | Delete own account               |
| DELETE | `/user/:id`              | Delete user (admin)              |

#### Admin

| Method | Path                                             | Purpose                        |
| ------ | ------------------------------------------------ | ------------------------------ |
| GET    | `/admin`                                         | Admin dashboard data           |
| GET    | `/admin/demo-user/sync`                          | Sync demo user                 |
| POST   | `/admin/gather`                                  | Gather data (7 days)           |
| POST   | `/admin/gather/max`                              | Gather data (max range)        |
| POST   | `/admin/gather/profile-data`                     | Gather all profile data        |
| POST   | `/admin/gather/profile-data/:dataSource/:symbol` | Gather profile data for symbol |
| POST   | `/admin/gather/:dataSource/:symbol`              | Gather symbol data             |
| POST   | `/admin/gather/:dataSource/:symbol/:dateString`  | Gather symbol data for date    |
| GET    | `/admin/market-data`                             | Market data list               |
| POST   | `/admin/market-data/:dataSource/:symbol/test`    | Test market data scraper       |
| POST   | `/admin/profile-data/:dataSource/:symbol`        | Add profile data               |
| DELETE | `/admin/profile-data/:dataSource/:symbol`        | Delete profile data            |
| PATCH  | `/admin/profile-data/:dataSource/:symbol`        | Update profile data            |
| PUT    | `/admin/settings/:key`                           | Update admin setting           |
| GET    | `/admin/user`                                    | List users                     |
| GET    | `/admin/user/:id`                                | Get user details               |

#### Admin Queue

| Method | Path                           | Purpose               |
| ------ | ------------------------------ | --------------------- |
| GET    | `/admin/queue/job`             | List jobs             |
| GET    | `/admin/queue/job/:id/execute` | Execute job           |
| DELETE | `/admin/queue/job`             | Delete jobs by status |
| DELETE | `/admin/queue/job/:id`         | Delete specific job   |

#### Cache

| Method | Path           | Purpose           |
| ------ | -------------- | ----------------- |
| POST   | `/cache/flush` | Flush Redis cache |

#### Symbol (remaining)

| Method | Path                                      | Purpose                  |
| ------ | ----------------------------------------- | ------------------------ |
| GET    | `/symbol/:dataSource/:symbol`             | Get symbol data          |
| GET    | `/symbol/:dataSource/:symbol/:dateString` | Get symbol data for date |

#### Exchange Rate

| Method | Path                                 | Purpose           |
| ------ | ------------------------------------ | ----------------- |
| GET    | `/exchange-rate/:symbol/:dateString` | Get exchange rate |

#### Import / Export

| Method | Path                                    | Purpose          |
| ------ | --------------------------------------- | ---------------- |
| GET    | `/export`                               | Export portfolio |
| POST   | `/import`                               | Import portfolio |
| GET    | `/import/dividends/:dataSource/:symbol` | Import dividends |

#### Market Data

| Method | Path                               | Purpose               |
| ------ | ---------------------------------- | --------------------- |
| GET    | `/market-data/markets`             | Market data overview  |
| GET    | `/market-data/:dataSource/:symbol` | Market data by symbol |
| POST   | `/market-data/:dataSource/:symbol` | Update market data    |

#### Data Provider (Ghostfolio)

| Method | Path                                               | Purpose         |
| ------ | -------------------------------------------------- | --------------- |
| GET    | `/data-providers/ghostfolio/asset-profile/:symbol` | Asset profile   |
| GET    | `/data-providers/ghostfolio/dividends/:symbol`     | Dividends       |
| GET    | `/data-providers/ghostfolio/historical/:symbol`    | Historical data |
| GET    | `/data-providers/ghostfolio/lookup`                | Symbol lookup   |
| GET    | `/data-providers/ghostfolio/quotes`                | Quotes          |
| GET    | `/data-providers/ghostfolio/status`                | Provider status |

#### Benchmarks (remaining)

| Method | Path                                               | Purpose               |
| ------ | -------------------------------------------------- | --------------------- |
| POST   | `/benchmarks`                                      | Add benchmark         |
| DELETE | `/benchmarks/:dataSource/:symbol`                  | Delete benchmark      |
| GET    | `/benchmarks/:dataSource/:symbol/:startDateString` | Benchmark market data |

#### Public / Infrastructure

| Method | Path                                     | Purpose               |
| ------ | ---------------------------------------- | --------------------- |
| GET    | `/public/:accessId/portfolio`            | Public portfolio view |
| GET    | `/logo/:dataSource/:symbol`              | Logo by symbol        |
| GET    | `/logo`                                  | Logo by URL           |
| GET    | `/asset/:dataSource/:symbol`             | Asset details         |
| GET    | `/sitemap.xml`                           | Sitemap               |
| GET    | `/ai/prompt/:mode`                       | AI prompt             |
| POST   | `/api-keys`                              | Create API key        |
| GET    | `/assets/:languageCode/site.webmanifest` | Web manifest          |
| GET    | `/health/data-enhancer/:name`            | Data enhancer health  |
| GET    | `/health/data-provider/:dataSource`      | Data provider health  |
| DELETE | `/auth-device/:id`                       | Delete auth device    |

#### Subscription

| Method | Path                                    | Purpose                |
| ------ | --------------------------------------- | ---------------------- |
| POST   | `/subscription/redeem-coupon`           | Redeem coupon          |
| GET    | `/subscription/stripe/callback`         | Stripe callback        |
| POST   | `/subscription/stripe/checkout-session` | Create Stripe checkout |

#### Agent

| Method | Path                 | Purpose                      |
| ------ | -------------------- | ---------------------------- |
| POST   | `/agent/chat`        | Chat with agent (SSE stream) |
| GET    | `/agent/session/:id` | Get session history          |
| POST   | `/agent/feedback`    | Submit feedback              |

### Query parameter patterns

Many endpoints accept a common set of filter query parameters:

- `accounts` — Comma-separated account IDs
- `assetClasses` — Comma-separated asset classes
- `dataSource` — Data source filter
- `symbol` — Symbol filter
- `tags` — Comma-separated tag IDs
- `range` — Date range (`1d`, `1y`, `5y`, `max`, `mtd`, `wtd`, `ytd`)
- `skip` / `take` — Pagination (admin endpoints)
- `sortColumn` / `sortDirection` — Sorting
- `query` — Search text
- `groupBy` — Grouping (`month`, `year`)
