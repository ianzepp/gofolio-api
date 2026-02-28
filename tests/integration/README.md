# Differential Integration Harness

This test suite runs the same HTTP requests against:

- legacy NestJS API (`LEGACY_BASE_URL`)
- Go API (`GO_BASE_URL`)

and compares responses per case.

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

Current fixture set (`tests/integration/testdata/cases.json`) has:

- `103` total cases
- `75` status-compare cases
- `28` JSON-compare cases
- `12` export-focused parity cases

## Auth

Cases can set `"auth"`:

- `"jwt"` -> uses `JWT_TOKEN` env var as Bearer token
- `"jwt:ENV_NAME"` -> uses that env var as Bearer token
- `"api-key"` -> uses `API_KEY` env var as `Api-Key` token
- `"api-key:ENV_NAME"` -> uses that env var as `Api-Key` token

Path, headers, and JSON body support env expansion via `${ENV_NAME}`.

## Case Fixture

Edit `tests/integration/testdata/cases.json`.

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
