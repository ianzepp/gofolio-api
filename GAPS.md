# Go API — Remaining Gaps

Tracked inventory of everything between current state and full parity with the NestJS backend. Each section is a workstream; items within are ordered by priority/dependency.

Last updated: 2026-03-01 (post-parity run)

---

## 1. Differential Baseline

Current differential baseline is fully green:

- `196/196` passing (`156` status + `40` JSON)
- Verified on `2026-03-01` with `./tests/integration/run.sh`
- Baseline env used for Go API in parity run:
  - `JWT_SECRET_KEY=1234`
  - `ACCESS_TOKEN_SALT=1234`
  - `DATABASE_URL=postgresql://user:1234@localhost:5432/ghostfolio-db?connect_timeout=300&sslmode=prefer`

Remaining parity work is now about unimplemented/stubbed in-scope routes and untested semantics, not currently failing differential fixtures.

---

## 2. Stubbed In-Scope Routes (~30 routes)

These routes return empty/mock JSON and are within the declared scope of the rewrite.

### 2a. Admin — Data Gather (6 routes)

Background job triggers. NestJS uses BullMQ to enqueue gather tasks.

| Method | Path                                             | Purpose                        |
| ------ | ------------------------------------------------ | ------------------------------ |
| POST   | `/admin/gather`                                  | Gather data (7 days)           |
| POST   | `/admin/gather/max`                              | Gather data (max range)        |
| POST   | `/admin/gather/profile-data`                     | Gather all profile data        |
| POST   | `/admin/gather/profile-data/:dataSource/:symbol` | Gather profile data for symbol |
| POST   | `/admin/gather/:dataSource/:symbol`              | Gather symbol data             |
| POST   | `/admin/gather/:dataSource/:symbol/:dateString`  | Gather symbol data for date    |

Status: routes now enqueue Redis-backed queue jobs. Remaining work is worker-side execution parity and payload semantics.

### 2b. Admin — Profile Data CRUD (3 routes) — 2/3 PARTIAL

Delete + update are wired through `AdminService`.
Create currently returns legacy-style `404` parity response and needs full implementation semantics.

### 2c. Admin — Misc (3 routes) — 1/3 PARTIAL

| Method | Path                                          | Purpose                  |
| ------ | --------------------------------------------- | ------------------------ |
| PUT    | `/admin/settings/:key`                        | Update admin setting     |
| GET    | `/admin/demo-user/sync`                       | Sync demo user           |
| POST   | `/admin/market-data/:dataSource/:symbol/test` | Test market data scraper |

Status: `POST /admin/market-data/:dataSource/:symbol/test` now enqueues queue jobs; settings and demo-user-sync are still stubs.

### ~~2d. Import (2 routes)~~ DONE

Wired: POST /import creates orders from Ghostfolio interchange JSON. GET /import/dividends returns existing dividend orders for a symbol.

### ~~2e. Market Data Write (1 route)~~ DONE

Wired: Bulk upsert via existing `UpsertMarketPrice` repo method.

### 2f. Benchmark Management (3 routes) — 2/3 DONE

Create and delete wired. Detail (GET with performance calc) remains a stub.

| Method | Path                                               | Status |
| ------ | -------------------------------------------------- | ------ |
| POST   | `/benchmarks`                                      | DONE   |
| DELETE | `/benchmarks/:dataSource/:symbol`                  | DONE   |
| GET    | `/benchmarks/:dataSource/:symbol/:startDateString` | Stub   |

### 2g. Portfolio — Remaining (2 routes) — 1/2 DONE

Holding tags update wired. Report remains a stub.

| Method | Path                                          | Status |
| ------ | --------------------------------------------- | ------ |
| GET    | `/portfolio/report`                           | Stub   |
| PUT    | `/portfolio/holding/:dataSource/:symbol/tags` | DONE   |

### ~~2h. Symbol Date Lookup (1 route)~~ DONE

Wired: Full implementation with date parsing and DB lookup.

## 3. Excluded from Scope (16 routes)

These are intentionally not being ported. Documented here for completeness.

| Area                         | Routes | Reason                                 |
| ---------------------------- | ------ | -------------------------------------- |
| `auth/oauth` (Google + OIDC) | 4      | Server-side OAuth provider integration |
| `auth/webauthn`              | 4      | FIDO2/WebAuthn crypto                  |
| `subscription/*`             | 3      | Stripe-only billing                    |
| `agent/*`                    | 3      | Local-only agent code, not upstream    |
| static/web endpoints         | 2      | API-only server (`/sitemap.xml`, `/assets/:languageCode/site.webmanifest`) |

---

### ~~4. README Drift~~ DONE

Main README parity snapshot updated to reflect 196 cases across 21 fixture files.

---

## 5. Test Coverage Gaps

Areas with no or minimal differential test coverage:

- [ ] Import endpoints (no dedicated fixtures)
- [ ] Admin gather endpoints (no JSON fixtures — enqueue semantics only)
- [ ] Admin profile data CRUD (minimal fixtures; create semantics not covered)
- [ ] Benchmark create/delete/detail (status-only coverage; no JSON fixtures)
- [ ] Market data create (no dedicated fixtures)
- [ ] Portfolio report (status-only coverage; handler is stub)
- [ ] Holding tags update (status-only coverage; no JSON fixtures)
- [ ] Symbol date lookup (no JSON fixture)

---

## 6. Correctness Issues (from go-correctness-surgeon audit)

Issues found by deep correctness analysis, ordered by severity.

### HIGH

None open. Previously reported HIGH issues were fixed.

### MEDIUM

| #   | Issue                                        | Location                     | Description                                                                                          |
| --- | -------------------------------------------- | ---------------------------- | ---------------------------------------------------------------------------------------------------- |
| 1   | Swallowed errors in `GetPublicPortfolio`     | `handler/public.go:59-61`    | Three `GetPerformance` calls discard errors with `_`. Silent zero-data on failure.                   |
| 2   | Double `buildCalculator` in `GetDetails`     | `service/portfolio.go:432`   | Calls `buildCalculator`+`computeSnapshot`, then calls `GetHoldings` which does it again. 2x DB load. |
| 3   | Double query in `GetDividends`               | `service/portfolio.go:524`   | `buildCalculator` loads all orders, then `GetDividends` queries them again separately.               |
| 4   | `TransferBalance` race condition             | `handler/account.go:360`     | Balance check outside transaction; concurrent transfers can overdraw.                                |
| 5   | `isCurrencyLikeSymbol` uses `strings.Replace` | `handler/market_data.go:160` | Should use `strings.TrimSuffix`; "USDJPY" misidentified as currency-like.                           |
| 6   | Yahoo forex detection too broad              | `provider/yahoo.go:621`      | 6-letter uppercase stock tickers treated as forex pairs and appended with `=X`.                      |
| 7   | No graceful shutdown                         | `main.go:324`                | No signal handling or `server.Shutdown()`; in-flight requests dropped on SIGTERM.                    |

### LOW

| #   | Issue                                   | Location                       | Description                                                                      |
| --- | --------------------------------------- | ------------------------------ | -------------------------------------------------------------------------------- |
| 1   | Errgroup panic leaks goroutine          | `service/data_provider.go:84`  | Panic in provider goroutine prevents channel close; request goroutine blocks forever. |
| 2   | Large dynamic SQL OR-chains             | `repository/market_data.go:207`| Many-symbol queries build huge OR-chains; consider CTE or temp table at scale.   |
| 3   | `GetAccounts` totalValue semantically wrong | `handler/account.go:100`   | Adds cost basis + dividends + interest; not actual portfolio value.               |

---

## 7. Suggested Priority Order

1. **Admin gather execution parity** — Redis queue enqueue is wired; worker-side execution semantics still need parity.
2. **Benchmark detail endpoint** — Implement `GET /benchmarks/:dataSource/:symbol/:startDateString`.
3. **Portfolio report** — Implement `GET /portfolio/report`.
4. **Admin settings + demo-user sync** — Replace remaining stubs with service-backed behavior.
5. **Profile data create semantics** — Replace legacy-404 parity stub with full implementation once parity target is defined.
6. **Coverage expansion** — Add dedicated fixtures for import, market-data write, profile-data CRUD, and benchmark detail JSON.
7. **Correctness fixes (MEDIUM/LOW)** — Performance double-queries, race conditions, graceful shutdown.
