# Migrate Off Redis — Pure Go Replacements

## Current State

- **No differential tests** cover cache or queue endpoints
- **Go has zero Redis usage** — both handlers are stubs returning `{}` or `[]`
- The NestJS side uses Redis for two things the Go side doesn't do yet:
  1. **TTL cache** for quote prices, portfolio snapshots, benchmarks
  2. **Bull job queue** for background data gathering and portfolio computation

## What We Actually Need (Pure Go)

### Cache Layer

A simple in-memory cache with TTL support. The API surface is tiny:

| Operation | Use Case |
|-----------|----------|
| `Get(key)` | Check for cached quotes/snapshots |
| `Set(key, value, ttl)` | Store with expiration |
| `Del(key...)` | Invalidate on portfolio changes |
| `Flush()` | Admin cache clear endpoint |

One file, ~80 lines. A `sync.Map` or `map` + `sync.RWMutex` with a background goroutine sweeping expired entries. Or just pull in `hashicorp/golang-lru/v2` if you want eviction too.

#### NestJS Cache Keys (for reference)

- `quote-{symbol}-{dataSource}` — quote prices, 1-min TTL
- `portfolio-snapshot-{userId}` or `portfolio-snapshot-{userId}-{filtersHash}` — infinite TTL, event-invalidated
- `BENCHMARKS` — manual expiration stored in value

### Job Queue

Goroutines + channels. The Go equivalent of Bull is literally:

| Bull Concept | Go Equivalent |
|-------------|---------------|
| Queue | Buffered `chan Job` |
| Worker | Goroutine reading from channel |
| Concurrency limit | Semaphore or worker pool size |
| Rate limiting | `time.Ticker` |
| Retry | Loop with backoff |
| Job state | Struct field (`pending`/`active`/`done`/`failed`) |

One file, ~150 lines. No persistence needed unless you want jobs to survive restarts (SQLite table would handle that later).

#### NestJS Job Types (for reference)

Two queues in NestJS:
1. **DATA_GATHERING_QUEUE** — async market data fetches from Yahoo etc.
   - `GATHER_ASSET_PROFILE` — fetch asset metadata
   - `GATHER_HISTORICAL_MARKET_DATA` — fetch historical prices
   - Rate-limited: 1 request / 4 seconds
   - Concurrency: 2
2. **PORTFOLIO_SNAPSHOT_COMPUTATION_QUEUE** — compute user portfolio metrics
   - Single job type
   - Concurrency: 2, lock duration: 300s

### Admin Queue Endpoints

The `GET /admin/queue/job` endpoint needs to return the `AdminJobs` shape:

```json
{
  "jobs": [
    {
      "attemptsMade": 0,
      "data": {},
      "finishedOn": 0,
      "id": "string",
      "name": "string",
      "opts": {},
      "stacktrace": "",
      "state": "waiting|completed|failed|delayed|active|stuck",
      "timestamp": 0
    }
  ]
}
```

The other three endpoints (delete jobs, delete job by ID, execute job by ID) operate on the same in-memory job list.

### Cache Flush Endpoint

`POST /api/v1/cache/flush` — calls `Flush()` on the in-memory cache. Returns empty body with 200.

## Implementation Plan

Two new files:
- `internal/cache/cache.go` (~80 lines) — in-memory TTL cache
- `internal/queue/queue.go` (~150 lines) — goroutine-based job queue

Wire into:
- `cmd/server/main.go` — instantiate cache and queue, pass into `Deps`
- `internal/handler/cache.go` — call `cache.Flush()`
- `internal/handler/admin_queue.go` — expose job list, delete/execute operations
- `internal/service/data_provider.go` — use cache for quote prices
- Future: portfolio snapshot caching, background data gathering jobs

## Why This Works

- **Single binary, single instance** — no need for distributed cache or cross-process job coordination
- **Go's concurrency model** — goroutines and channels replace Bull entirely
- **No external dependencies** — zero infrastructure beyond the binary itself
- **API parity** — same HTTP endpoints, same JSON shapes, no Redis
