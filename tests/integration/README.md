# Differential Integration Harness

This test suite runs the same HTTP requests against:

- legacy NestJS API (`LEGACY_BASE_URL`)
- Go API (`GO_BASE_URL`)

and compares responses per case.

## Run

```bash
cd apps/api-go
LEGACY_BASE_URL=http://localhost:3333 \
GO_BASE_URL=http://localhost:3334 \
go test ./tests/integration -v
```

If base URLs are unset, defaults are used:

- `LEGACY_BASE_URL=http://localhost:3333`
- `GO_BASE_URL=http://localhost:3334`

If either API is unreachable at `/api/v1/health`, tests are skipped.

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
- `auth`: `""`, `"jwt"`, or `"api-key"`
- `auth`: `""`, `"jwt"`, `"jwt:ENV_NAME"`, `"api-key"`, or `"api-key:ENV_NAME"`
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
