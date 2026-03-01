# Differential Integration Harness

This test suite runs the same HTTP requests against:

- legacy NestJS API (`LEGACY_BASE_URL`)
- Go API (`GO_BASE_URL`)

and compares responses per case.

Latest verified full run: `2026-03-01`, `196/196` passing with `./tests/integration/run.sh`.

## Run

Preferred (auto-start `api-go`, preflight checks, auto-cleanup):

```bash
cd apps/api-go
./tests/integration/run.sh
```

Manual:

```bash
cd apps/api-go
LEGACY_BASE_URL=http://127.0.0.1:3333 \
GO_BASE_URL=http://127.0.0.1:3334 \
go test ./tests/integration -v
```

If base URLs are unset, defaults are used:

- `LEGACY_BASE_URL=http://127.0.0.1:3333`
- `GO_BASE_URL=http://127.0.0.1:3334`

If either API is unreachable at `/api/v1/health`, tests are skipped.

### `run.sh` behavior

- Verifies legacy API is up before running tests.
- Bootstraps a fresh legacy user by default:
  - calls `POST /api/v1/user`
  - uses returned `authToken` (or exchanges `accessToken` via `POST /api/v1/auth/anonymous`)
  - exports `JWT_USER_TOKEN`, `JWT_TOKEN`, `LEGACY_USER_ID` for differential cases
  - creates a second throwaway user and exports `JWT_EMPTY_USER_TOKEN`,
    `EMPTY_USER_ID`, `EMPTY_USER_ACCESS_TOKEN` for destructive/edge cases
  - creates transfer test accounts for the bootstrapped user and exports
    `TRANSFER_ACCOUNT_FROM_ID` and `TRANSFER_ACCOUNT_TO_ID`
  - seeds two deterministic transfer-account orders and exports
    `SEED_ORDER_ID` and `SEED_ORDER_TWO_ID` for export filter parity cases
  - exports `ADMIN_USER_ID` as a guaranteed foreign UUID for ownership-negative cases
  - attempts DB-assisted admin bootstrap and exports `JWT_ADMIN_TOKEN` +
    `ADMIN_DB_USER_ID` for admin-control parity cases when available
  - attempts to load `demoAuthToken` from `GET /api/v1/info` and exports
    `JWT_DEMO_TOKEN` + `DEMO_USER_ID` when available
  - attempts to create/export `API_KEY_USER` via `POST /api/v1/api-keys` when permitted
  - if that fails, attempts DB-assisted fallback:
    - query admin user id from PostgreSQL
    - sign an admin JWT with `JWT_SECRET_KEY`
    - call `POST /api/v1/api-keys` and export `API_KEY_USER`
  - if API key creation is unavailable, falls back to a dummy `API_KEY_USER`
    value so unauthorized API-key parity checks still run
  - if `/info` has no `demoAuthToken`, attempts DB-assisted demo bootstrap
    and exports `JWT_DEMO_TOKEN` + `DEMO_USER_ID`
  - if no demo user exists, creates a throwaway user and updates its role to
    `DEMO` for parity testing
  - creates an isolated ROAI test user with deterministic orders (BTC-USD, MSFT)
    seeded via direct DB inserts and exports `JWT_ROAI_TOKEN`, `ROAI_USER_ID`,
    `ROAI_ACCOUNT_ID`, `ROAI_BTCUSD_ORDER_ID`, `ROAI_MSFT_BUY_ORDER_ID`,
    and `ROAI_SEEDED` for ROAI calculator parity cases
  - inserts one throwaway `AuthDevice` row and exports `AUTH_DEVICE_ID` for
    `DELETE /api/v1/auth-device/{id}` parity checks
- Builds a persistent Go binary and starts it on `GO_PORT` (default `3334`) if it is not already running.
- Waits for Go health endpoint before running tests.
- Automatically stops the Go API process it started.
- Writes Go API startup/runtime logs to `${TMPDIR:-/tmp}/api-go-integration.log`.

Useful env vars:

- `LEGACY_BASE_URL` (default `http://127.0.0.1:3333`)
- `GO_BASE_URL` (default `http://127.0.0.1:3334`)
- `GO_PORT` (default `3334`)
- `GO_BINARY_PATH` (default `apps/api-go/bin/api-go`)
- `START_TIMEOUT_SECONDS` (default `20`)
- `BOOTSTRAP_LEGACY_USER` (`1` default; set `0` to disable auto user bootstrap)

For JWT-auth parity cases, Go must use the same `JWT_SECRET_KEY` as legacy.
`run.sh` resolves it in this order:

1. `JWT_SECRET_KEY` from current shell env
2. `JWT_SECRET_KEY` from repo `.env` (if present)

For DB-backed parity cases, Go also needs `DATABASE_URL`.
`run.sh` resolves it in this order:

1. `DATABASE_URL` from current shell env
2. `DATABASE_URL` from repo `.env` (if present)

For access-token parity cases, Go also needs `ACCESS_TOKEN_SALT`.
`run.sh` resolves it in this order:

1. `ACCESS_TOKEN_SALT` from current shell env
2. `ACCESS_TOKEN_SALT` from repo `.env` (if present)

Optional tools used by advanced bootstrap paths:

- `jq` (required by runner)
- `openssl` (for admin JWT signing fallback)
- `psql` (for admin user lookup fallback)

## Current Coverage

Fixture files live in `tests/integration/testdata/` as JSON arrays (primarily
one file per handler, plus cross-cutting files like `new_endpoints.json`). The
harness auto-loads all `*.json` files in that directory.

Current totals across all fixture files:

- `196` total cases
- `156` status-compare cases
- `40` JSON-compare cases
- `21` fixture files
- HTTP client timeout: `60s` per request (legacy can be slow on some seeded
  portfolio filter cases)

## Fixture Catalog

Each fixture file contains an array of test-case objects. Case names and modes:

### `access.json` (4)

- `access-delete-foreign-id-forbidden-parity` (`status`)
- `access-update-foreign-id-forbidden-parity` (`status`)
- `access-get-no-auth-unauthorized` (`status`)
- `access-get-user-json-parity` (`json`)

### `account.json` (12)

- `account-transfer-missing-accounts-not-found-parity` (`status`)
- `account-transfer-same-account-bad-request-parity` (`status`)
- `account-transfer-insufficient-funds-bad-request-parity` (`status`)
- `account-get-no-auth-unauthorized` (`status`)
- `account-get-no-auth-unauthorized-json-parity` (`json`)
- `account-transfer-no-auth-unauthorized` (`status`)
- `account-get-by-id-no-auth-unauthorized` (`status`)
- `account-get-by-id-user-ok-parity` (`status`)
- `account-get-by-id-user-json-parity` (`json`)
- `account-balances-no-auth-unauthorized` (`status`)
- `account-balances-user-ok-parity` (`status`)
- `account-get-json-parity` (`json`)

### `account_balance.json` (5)

- `account-balance-post-no-auth-unauthorized` (`status`)
- `account-balance-delete-no-auth-unauthorized` (`status`)
- `account-balance-post-foreign-account-forbidden-parity` (`status`)
- `account-balance-delete-foreign-id-forbidden-parity` (`status`)
- `account-balance-post-user-created-parity` (`status`)

### `admin.json` (16)

- `admin-get-no-auth-unauthorized` (`status`)
- `admin-get-user-forbidden-parity` (`status`)
- `admin-get-user-forbidden-json-parity` (`json`)
- `admin-get-admin-ok-parity` (`status`)
- `admin-market-data-no-auth-unauthorized` (`status`)
- `admin-market-data-user-forbidden-parity` (`status`)
- `admin-market-data-admin-ok-parity` (`status`)
- `admin-market-data-admin-query-json-parity` (`json`)
- `admin-market-data-admin-sort-json-parity` (`json`)
- `admin-users-no-auth-unauthorized` (`status`)
- `admin-users-user-forbidden-parity` (`status`)
- `admin-users-admin-ok-parity` (`status`)
- `admin-user-by-id-no-auth-unauthorized` (`status`)
- `admin-user-by-id-admin-ok-parity` (`status`)
- `admin-user-by-id-admin-missing-not-found-parity` (`status`)
- `admin-user-by-id-admin-invalid-format-parity` (`status`)

### `asset.json` (3)

- `asset-get-status-parity` (`status`)
- `asset-get-missing-symbol-json-parity` (`json`)
- `asset-get-missing-symbol-fallback-parity` (`status`)

### `benchmark.json` (6)

- `benchmarks-post-no-auth-unauthorized` (`status`)
- `benchmarks-post-user-forbidden-parity` (`status`)
- `benchmarks-delete-no-auth-unauthorized` (`status`)
- `benchmarks-detail-no-auth-unauthorized` (`status`)
- `benchmarks-get-status-parity` (`status`)
- `benchmarks-get-json-parity` (`json`)

### `data_provider.json` (8)

- `data-provider-ghostfolio-asset-profile-no-auth-unauthorized` (`status`)
- `data-provider-ghostfolio-dividends-no-auth-unauthorized` (`status`)
- `data-provider-ghostfolio-historical-no-auth-unauthorized` (`status`)
- `data-provider-ghostfolio-lookup-no-auth-unauthorized` (`status`)
- `data-provider-ghostfolio-quotes-no-auth-unauthorized` (`status`)
- `data-provider-ghostfolio-status-no-auth-unauthorized` (`status`)
- `data-provider-ghostfolio-lookup-empty-query-status-parity` (`status`)
- `data-provider-ghostfolio-status-api-key-status-parity` (`status`)

### `exchange_rate.json` (3)

- `exchange-rate-no-auth-unauthorized` (`status`)
- `exchange-rate-auth-status-parity` (`status`)
- `exchange-rate-unknown-pair-status-parity` (`status`)

### `export.json` (12)

- `export-no-auth-unauthorized` (`status`)
- `export-no-auth-unauthorized-json-parity` (`json`)
- `export-user-ok-parity` (`status`)
- `export-user-filtered-account-json-parity` (`json`)
- `export-user-filtered-symbol-json-parity` (`json`)
- `export-user-filtered-activity-id-json-parity` (`json`)
- `export-user-filtered-account-and-activity-id-json-parity` (`json`)
- `export-user-filtered-data-source-json-parity` (`json`)
- `export-user-filtered-activity-ids-multi-json-parity` (`json`)
- `export-user-filtered-symbol-msft-json-parity` (`json`)
- `export-user-filtered-account-to-and-activity-id-json-parity` (`json`)
- `export-user-filtered-account-and-activity-mismatch-json-parity` (`json`)

### `health.json` (3)

- `health-basic-status` (`status`)
- `health-data-enhancer-status` (`status`)
- `health-data-provider-status` (`status`)

### `info.json` (2)

- `info-status` (`status`)
- `info-json-parity` (`json`)

### `logo.json` (3)

- `logo-by-url-status-parity` (`status`)
- `logo-by-url-missing-url-status-parity` (`status`)
- `logo-by-symbol-unknown-status-parity` (`status`)

### `market_data.json` (7)

- `market-data-by-symbol-no-auth-unauthorized` (`status`)
- `market-data-by-symbol-no-auth-unauthorized-json-parity` (`json`)
- `market-data-by-symbol-user-common-forbidden-parity` (`status`)
- `market-data-by-symbol-user-common-forbidden-json-parity` (`json`)
- `market-data-by-symbol-admin-common-ok-parity` (`status`)
- `market-data-by-symbol-missing-noncurrency-not-found-parity` (`status`)
- `market-data-by-symbol-missing-currency-fallback-parity` (`status`)

### `new_endpoints.json` (27)

- `auth-anonymous-post-valid-json-parity` (`json`)
- `auth-anonymous-post-invalid-forbidden-parity` (`status`)
- `auth-anonymous-get-valid-json-parity` (`json`)
- `auth-anonymous-get-invalid-forbidden-parity` (`status`)
- `api-keys-post-no-auth-unauthorized` (`status`)
- `api-keys-post-user-json-parity` (`json`)
- `ai-prompt-portfolio-no-auth-unauthorized` (`status`)
- `ai-prompt-portfolio-user-status-parity` (`status`)
- `ai-prompt-analysis-user-status-parity` (`status`)
- `cache-flush-no-auth-unauthorized` (`status`)
- `cache-flush-user-forbidden-parity` (`status`)
- `cache-flush-admin-status-parity` (`status`)
- `auth-device-delete-no-auth-unauthorized` (`status`)
- `auth-device-delete-user-status-parity` (`status`)
- `public-portfolio-missing-not-found-parity` (`status`)
- `admin-queue-job-get-no-auth-unauthorized` (`status`)
- `admin-queue-job-get-user-forbidden-parity` (`status`)
- `admin-queue-job-get-admin-status-parity` (`status`)
- `admin-queue-job-delete-no-auth-unauthorized` (`status`)
- `admin-queue-job-delete-user-forbidden-parity` (`status`)
- `admin-queue-job-delete-admin-status-parity` (`status`)
- `admin-queue-job-id-delete-no-auth-unauthorized` (`status`)
- `admin-queue-job-id-delete-user-forbidden-parity` (`status`)
- `admin-queue-job-id-delete-admin-status-parity` (`status`)
- `admin-queue-job-execute-no-auth-unauthorized` (`status`)
- `admin-queue-job-execute-user-forbidden-parity` (`status`)
- `admin-queue-job-execute-admin-status-parity` (`status`)

### `order.json` (8)

- `order-delete-empty-user-json-parity` (`json`)
- `order-get-foreign-id-not-found-parity` (`status`)
- `order-delete-foreign-id-forbidden-parity` (`status`)
- `order-update-foreign-id-forbidden-parity` (`status`)
- `order-get-no-auth-unauthorized` (`status`)
- `order-get-no-auth-unauthorized-json-parity` (`json`)
- `order-delete-no-auth-unauthorized` (`status`)
- `order-get-json-parity` (`json`)

### `platform.json` (10)

- `platforms-get-user-ok-parity` (`status`)
- `platform-get-user-forbidden-parity` (`status`)
- `platform-get-user-forbidden-json-parity` (`json`)
- `platforms-get-no-auth-unauthorized` (`status`)
- `platform-get-no-auth-unauthorized` (`status`)
- `platform-create-user-forbidden-parity` (`status`)
- `platform-update-user-forbidden-parity` (`status`)
- `platform-delete-user-forbidden-parity` (`status`)
- `platform-update-admin-foreign-id-forbidden-parity` (`status`)
- `platform-delete-admin-foreign-id-forbidden-parity` (`status`)

### `portfolio.json` (38)

- `portfolio-holdings-authed` (`status`)
- `portfolio-holdings-unauthed` (`status`)
- `portfolio-details-authed` (`status`)
- `portfolio-details-unauthed` (`status`)
- `portfolio-details-range-1y` (`status`)
- `portfolio-details-range-ytd` (`status`)
- `portfolio-performance-authed` (`status`)
- `portfolio-performance-unauthed` (`status`)
- `portfolio-performance-range-1d` (`status`)
- `portfolio-performance-range-1y` (`status`)
- `portfolio-performance-range-ytd` (`status`)
- `portfolio-dividends-authed` (`status`)
- `portfolio-dividends-unauthed` (`status`)
- `portfolio-investments-authed` (`status`)
- `portfolio-investments-unauthed` (`status`)
- `portfolio-investments-group-by-month` (`status`)
- `portfolio-investments-group-by-year` (`status`)
- `portfolio-report-authed` (`status`)
- `portfolio-report-unauthed` (`status`)
- `portfolio-holding-aapl-authed` (`status`)
- `portfolio-holding-msft-authed` (`status`)
- `portfolio-holding-unknown-symbol` (`status`)
- `portfolio-holding-unauthed` (`status`)
- `portfolio-holdings-filter-by-account` (`status`)
- `portfolio-details-filter-by-account` (`status`)
- `portfolio-performance-filter-by-account` (`status`)
- `portfolio-holdings-empty-user` (`status`)
- `portfolio-details-empty-user` (`status`)
- `portfolio-performance-empty-user` (`status`)
- `portfolio-holding-tags-update-unauthed` (`status`)
- `roai-holdings-json` (`json`)
- `roai-details-json` (`json`)
- `roai-performance-max-json` (`json`)
- `roai-holding-btcusd-json` (`json`)
- `roai-holding-msft-json` (`json`)
- `roai-dividends-json` (`json`)
- `roai-investments-json` (`json`)
- `roai-holdings-filter-roai-account-json` (`json`)

### `symbol.json` (4)

- `symbol-lookup-no-auth-unauthorized` (`status`)
- `symbol-by-date-no-auth-unauthorized` (`status`)
- `symbol-get-invalid-data-source-not-found-parity` (`status`)
- `symbol-get-missing-symbol-not-found-parity` (`status`)

### `tag.json` (7)

- `tags-get-no-auth-unauthorized` (`status`)
- `tags-get-invalid-bearer-unauthorized` (`status`)
- `tags-get-api-key-user-parity` (`status`)
- `tags-get-user-json-parity` (`json`)
- `tags-post-user-missing-user-id-parity` (`status`)
- `tags-post-user-foreign-user-id-parity` (`status`)
- `tags-post-demo-forbidden-parity` (`status`)

### `user.json` (8)

- `user-access-token-wrong-token-forbidden-parity` (`status`)
- `user-delete-own-wrong-token-forbidden-parity` (`status`)
- `user-admin-access-token-existing-user-created-parity` (`status`)
- `user-admin-access-token-foreign-id-parity` (`status`)
- `user-signup-public-created-parity` (`status`)
- `user-get-no-auth-unauthorized` (`status`)
- `user-delete-self-by-id-admin-forbidden-parity` (`status`)
- `user-get-json-parity` (`json`)

### `watchlist.json` (10)

- `watchlist-get-no-auth-unauthorized` (`status`)
- `watchlist-post-no-auth-unauthorized` (`status`)
- `watchlist-delete-no-auth-unauthorized` (`status`)
- `watchlist-get-no-auth-unauthorized` (`status`)
- `watchlist-get-auth-status-parity` (`status`)
- `watchlist-get-empty-user-json-parity` (`json`)
- `watchlist-create-no-auth-unauthorized` (`status`)
- `watchlist-delete-no-auth-unauthorized` (`status`)
- `watchlist-create-unknown-symbol-bad-request-parity` (`status`)
- `watchlist-delete-unknown-symbol-not-found-parity` (`status`)

## Auth

Cases can set `"auth"`:

- `"jwt"` -> uses `JWT_TOKEN` env var as Bearer token
- `"jwt:ENV_NAME"` -> uses that env var as Bearer token
- `"api-key"` -> uses `API_KEY` env var as `Api-Key` token
- `"api-key:ENV_NAME"` -> uses that env var as `Api-Key` token

Path, headers, and JSON body support env expansion via `${ENV_NAME}`.

## Case Fixture

Add or edit a JSON file in `tests/integration/testdata/` (handler-specific or
cross-cutting, e.g. `portfolio.json` or `new_endpoints.json`).

Fields:

- `name` (required)
- `method` (required)
- `path` (required)
- `mode`: `"status"` or `"json"` (default `"status"`)
- `headers`: optional header map
- `auth`: `""`, `"jwt"`, `"jwt:ENV_NAME"`, `"api-key"`, or `"api-key:ENV_NAME"`
- `requiredEnv`: optional list of env vars required for the case; case is skipped if any are missing
- `body`: optional JSON payload
- `ignoreJsonPaths`: optional list of dot-paths to ignore in JSON mode

In JSON mode, the harness:

- decodes JSON from both responses
- removes all configured `ignoreJsonPaths`
- canonicalizes with Go `json.Marshal`
- compares byte-for-byte

Path syntax supports object traversal and arrays, including wildcard array
elements with `*` (for example `positions.*.id`).

Example JSON compare case:

```json
{
  "name": "public-portfolio-json",
  "method": "GET",
  "path": "/api/v1/public/abc123/portfolio",
  "mode": "json",
  "ignoreJsonPaths": ["meta.generatedAt", "positions.*.id"]
}
```
