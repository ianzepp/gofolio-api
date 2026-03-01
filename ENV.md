# Environment Variables

This file documents environment variables currently used by `gofolio-api`.

## Runtime / Deployment

| Variable | Required | Default | Used In | Notes |
| --- | --- | --- | --- | --- |
| `DATABASE_URL` | Yes (for DB-backed mode) | none | `internal/db/db.go` | Postgres DSN. If missing, server still starts in degraded mode (DB-backed endpoints return service unavailable). Startup normalizes `sslmode=prefer` to `sslmode=disable`. |
| `DB_CONNECT_TIMEOUT` | No | `10s` | `internal/db/db.go` | Duration for startup DB ping timeout parsing; invalid values fall back to default. |
| `DB_MAX_CONNS` | No | `10` | `internal/db/db.go` | Max open connections in pool; invalid values fall back to default. |
| `DB_MIN_CONNS` | No | `1` | `internal/db/db.go` | Maps to `SetMaxIdleConns`; invalid values fall back to default. |
| `DB_MAX_CONN_LIFETIME` | No | `1h` | `internal/db/db.go` | Connection max lifetime; invalid values fall back to default. |
| `DB_MAX_CONN_IDLE_TIME` | No | `15m` | `internal/db/db.go` | Connection idle timeout; invalid values fall back to default. |
| `PORT` | No | `3333` | `cmd/server/main.go` | HTTP listen port. |
| `JWT_SECRET_KEY` | Yes (for auth) | none | `cmd/server/main.go`, handlers | Required to sign/validate JWTs. If empty, authenticated requests fail. |
| `ACCESS_TOKEN_SALT` | Yes (for anonymous token flows) | none | `internal/handler/user.go` | Required to hash access tokens (HMAC-SHA512). |
| `BOOTSTRAP_ADMIN_TOKEN` | Required when no admin user exists in production | none | `internal/db/migrate.go` | One-time bootstrap admin token. Startup hashes it with `ACCESS_TOKEN_SALT` and inserts an `ADMIN` user if none exists. |
| `BOOTSTRAP_ADMIN_ID` | No | `00000000-0000-0000-0000-000000000001` | `internal/db/migrate.go` | Optional fixed ID for the bootstrap admin row. |
| `ROOT_URL` | No | `http://localhost:4200` | `internal/handler/auth.go` | Base frontend URL for OAuth callback redirects. |
| `ENABLE_FEATURE_AUTH_OIDC` | No | effectively `false` | `internal/handler/auth.go`, `internal/service/info.go` | Controls OIDC auth exposure. Handler checks strict string equality (`"true"`). |
| `ENABLE_FEATURE_SUBSCRIPTION` | No | effectively `false` | `internal/handler/public.go`, `internal/service/info.go` | Enables subscription-gated behavior and `info` global permission. Handler checks strict string equality (`"true"`). |
| `ENABLE_FEATURE_AUTH_TOKEN` | No | `true` | `internal/service/info.go` | Included in global permissions when enabled. |
| `ENABLE_FEATURE_STATISTICS` | No | `false` | `internal/service/info.go` | Included in global permissions when enabled. |
| `ENABLE_FEATURE_SYSTEM_MESSAGE` | No | `false` | `internal/service/info.go` | Included in global permissions when enabled. |
| `ENABLE_FEATURE_AUTH_GOOGLE` | No | `false` | `internal/service/info.go` | Included in global permissions when enabled. |
| `ENABLE_FEATURE_FEAR_AND_GREED_INDEX` | No | `false` | `internal/service/info.go` | Included in global permissions when enabled. |
| `GHOSTFOLIO_VERSION` | No | empty string | `internal/service/admin.go` | Exposed in admin dashboard payload as `version`. |

For `internal/service/info.go`, truthy feature-flag values are: `1`, `true`, `yes`, `on` (case-insensitive).

## Migration Bootstrap Behavior

On startup, if DB connection and ping succeed, the server checks for `public."User"`.

- If `public."User"` exists: schema migration is skipped.
- If missing: it executes the embedded initial schema SQL (`internal/db/migrations/000001_initial_schema.up.sql`).
- Seed SQL (`internal/db/migrations/000002_seed.up.sql`) runs on every startup.
- After schema/seed, startup ensures at least one `ADMIN` user exists:
  - If an admin already exists: no-op.
  - If no admin exists and `BOOTSTRAP_ADMIN_TOKEN` is set: creates admin user using hashed token.
  - If no admin exists and token is missing in production (`RAILWAY_ENVIRONMENT` set, or `APP_ENV`/`GO_ENV`/`NODE_ENV`/`ENVIRONMENT` = `production`): startup migration fails.
- If migration/bootstrap fails: DB is closed and server continues in degraded mode.

There is currently no env flag to disable startup migration.

## Integration Test Variables

These are used by `tests/integration` and are not required for normal server runtime.

| Variable | Required | Default | Notes |
| --- | --- | --- | --- |
| `LEGACY_BASE_URL` | Yes (for differential tests) | none | Base URL for legacy API under comparison. |
| `GO_BASE_URL` | Yes (for differential tests) | none | Base URL for this Go API under comparison. |
| `JWT_TOKEN` | Optional by fixture auth mode | none | Bearer token for JWT-auth test cases. |
| `API_KEY` | Optional by fixture auth mode | none | Bearer token for API-key-auth test cases. |
| `Any fixture-declared env` | Optional by fixture auth mode | none | Fixtures can require arbitrary env vars via `requiredEnv`, or use `auth` modes like `jwt:MY_ENV` / `api-key:MY_ENV`. |

## Example `.env`

```dotenv
DATABASE_URL=postgresql://user:password@localhost:5432/ghostfolio?sslmode=disable
PORT=3333
JWT_SECRET_KEY=replace-me
ACCESS_TOKEN_SALT=replace-me
BOOTSTRAP_ADMIN_TOKEN=replace-with-long-random-secret
BOOTSTRAP_ADMIN_ID=00000000-0000-0000-0000-000000000001
ROOT_URL=http://localhost:4200

DB_CONNECT_TIMEOUT=10s
DB_MAX_CONNS=10
DB_MIN_CONNS=1
DB_MAX_CONN_LIFETIME=1h
DB_MAX_CONN_IDLE_TIME=15m

ENABLE_FEATURE_AUTH_TOKEN=true
ENABLE_FEATURE_STATISTICS=false
ENABLE_FEATURE_SUBSCRIPTION=false
ENABLE_FEATURE_SYSTEM_MESSAGE=false
ENABLE_FEATURE_AUTH_GOOGLE=false
ENABLE_FEATURE_AUTH_OIDC=false
ENABLE_FEATURE_FEAR_AND_GREED_INDEX=false

GHOSTFOLIO_VERSION=dev
```
