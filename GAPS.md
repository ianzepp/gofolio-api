# Go API — Remaining Gaps

Tracked inventory of everything between current state and full parity with the NestJS backend. Each section is a workstream; items within are ordered by priority/dependency.

Last updated: 2026-03-01 (final push)

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

### ~~2c. Admin — Misc (3 routes)~~ 2/3 DONE

| Method | Path                                          | Purpose                  | Status |
| ------ | --------------------------------------------- | ------------------------ | ------ |
| PUT    | `/admin/settings/:key`                        | Update admin setting     | DONE   |
| GET    | `/admin/demo-user/sync`                       | Sync demo user           | Stub   |
| POST   | `/admin/market-data/:dataSource/:symbol/test` | Test market data scraper | DONE   |

Status: `PUT /admin/settings/:key` now upserts/deletes Property rows with read-only-mode side-effect. `POST .../test` enqueues queue jobs. Demo-user sync remains a stub.

### ~~2d. Import (2 routes)~~ DONE

Wired: POST /import creates orders from Ghostfolio interchange JSON. GET /import/dividends returns existing dividend orders for a symbol.

### ~~2e. Market Data Write (1 route)~~ DONE

Wired: Bulk upsert via existing `UpsertMarketPrice` repo method.

### ~~2f. Benchmark Management (3 routes)~~ DONE

| Method | Path                                               | Status |
| ------ | -------------------------------------------------- | ------ |
| POST   | `/benchmarks`                                      | DONE   |
| DELETE | `/benchmarks/:dataSource/:symbol`                  | DONE   |
| GET    | `/benchmarks/:dataSource/:symbol/:startDateString` | DONE   |

### ~~2g. Portfolio — Remaining (2 routes)~~ DONE

| Method | Path                                          | Status |
| ------ | --------------------------------------------- | ------ |
| GET    | `/portfolio/report`                           | DONE   |
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
- [ ] Benchmark detail (status-only coverage; no JSON fixtures)
- [ ] Market data create (no dedicated fixtures)
- [ ] Holding tags update (status-only coverage; no JSON fixtures)
- [ ] Symbol date lookup (no JSON fixture)

---

## 6. Correctness Issues (from go-correctness-surgeon audit)

Issues found by deep correctness analysis, ordered by severity.

### HIGH

None open. Previously reported HIGH issues were fixed.

### MEDIUM

| #   | Issue                                        | Location                     | Status |
| --- | -------------------------------------------- | ---------------------------- | ------ |
| ~~1~~   | ~~Swallowed errors in `GetPublicPortfolio`~~     | ~~`handler/public.go:59-61`~~    | **FIXED** — errors now logged |
| 2   | Double `buildCalculator` in `GetDetails`     | `service/portfolio.go:432`   | Open — perf optimization, not correctness |
| 3   | Double query in `GetDividends`               | `service/portfolio.go:524`   | Open — perf optimization, not correctness |
| ~~4~~   | ~~`TransferBalance` race condition~~             | ~~`handler/account.go:360`~~     | **FIXED** — SELECT FOR UPDATE inside tx |
| ~~5~~   | ~~`isCurrencyLikeSymbol` uses `strings.Replace`~~ | ~~`handler/market_data.go:160`~~ | **FIXED** — uses `strings.TrimSuffix` |
| ~~6~~   | ~~Yahoo forex detection too broad~~              | ~~`provider/yahoo.go:621`~~      | **FIXED** — known crypto base list |
| ~~7~~   | ~~No graceful shutdown~~                         | ~~`main.go:324`~~                | **FIXED** — SIGTERM/SIGINT + 15s drain |

### LOW

| #   | Issue                                   | Location                       | Status |
| --- | --------------------------------------- | ------------------------------ | ------ |
| ~~1~~   | ~~Errgroup panic leaks goroutine~~          | ~~`service/data_provider.go:84`~~  | **FIXED** — defer/recover in goroutine |
| 2   | Large dynamic SQL OR-chains             | `repository/market_data.go:207`| Open — perf optimization at scale |
| 3   | `GetAccounts` totalValue semantically wrong | `handler/account.go:100`   | Open — display-only, non-blocking |

---

## 7. Remaining Work (post-48h)

1. **Admin gather worker execution** — Redis queue enqueue is wired; worker-side execution semantics still need parity.
2. **Admin demo-user sync** — Stub, low priority.
3. **Profile data create semantics** — Replace legacy-404 parity stub with full implementation.
4. **Performance optimizations** — Double `buildCalculator` calls (MEDIUM #2, #3).
5. **Coverage expansion** — Add dedicated fixtures for import, market-data write, profile-data CRUD, and benchmark detail JSON.
6. **Regional/economic market rules** — Portfolio report currently passes regional/economic rules by default; needs country-level holding data for real evaluation.
