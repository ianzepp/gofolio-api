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
  - exports `JWT_USER_TOKEN` and `JWT_TOKEN` for differential cases
- Starts Go API on `GO_PORT` (default `3334`) if it is not already running.
- Waits for Go health endpoint before running tests.
- Automatically stops the Go API process it started.
- Writes Go API startup/runtime logs to `${TMPDIR:-/tmp}/api-go-integration.log`.

Useful env vars:

- `LEGACY_BASE_URL` (default `http://127.0.0.1:3333`)
- `GO_BASE_URL` (default `http://127.0.0.1:3334`)
- `GO_PORT` (default `3334`)
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
