// Package db provides the Postgres connection layer for api-go.
//
// It wraps pgxpool to give callers three things:
//
//  1. A DBTX interface so repositories work identically inside or outside a
//     transaction without needing separate code paths.
//  2. A Config struct loaded from environment variables with safe defaults,
//     so pool tuning is operator-controlled without a recompile.
//  3. Lifecycle helpers (Open, Ping) that fail fast during startup with
//     well-formatted errors.
//
// Connection pool sizing defaults (overridable via env):
//   - DB_MAX_CONNS=10    — ceiling on simultaneous Postgres connections
//   - DB_MIN_CONNS=1     — keep at least one warm connection at idle
//   - DB_MAX_CONN_LIFETIME=1h   — recycle connections to survive PgBouncer restarts
//   - DB_MAX_CONN_IDLE_TIME=15m — evict idle connections before server-side timeout
//   - DB_HEALTH_CHECK_PERIOD=1m — background ping cadence for stale detection
package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the minimal Postgres execution interface shared by *pgxpool.Pool
// and pgx.Tx. Repositories accept DBTX rather than a concrete type so that:
//   - Normal calls run against the pool (connection auto-managed).
//   - Transactional calls pass a pgx.Tx without changing repository code.
//
// TRADE-OFF: This surface area is intentionally small (Exec, Query, QueryRow).
// Richer pgxpool features (CopyFrom, SendBatch) require a type assertion.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx runs fn inside a serialized database transaction, committing on
// success and rolling back on any error. The deferred Rollback is a no-op
// after a successful Commit, making it safe to defer unconditionally.
//
// WHY: Centralizing transaction management here prevents callers from
// forgetting the rollback path, which historically causes connection leaks
// under pgxpool when a transaction is abandoned.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after commit

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const (
	defaultConnectTimeout = 10 * time.Second
)

// Config holds all tunable parameters for the Postgres connection pool.
// All fields except DatabaseURL have safe defaults applied by LoadConfigFromEnv,
// so callers do not need to set them explicitly in most environments.
type Config struct {
	DatabaseURL       string
	ConnectTimeout    time.Duration
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// LoadConfigFromEnv reads pool configuration from environment variables,
// applying documented defaults for optional fields. It returns an error only
// when DATABASE_URL is absent, since that is the only required value.
//
// EDGE: Invalid duration/integer values in optional env vars are silently
// replaced with defaults rather than failing startup — intentional so that
// a typo in DB_MAX_CONNS does not bring down the server.
func LoadConfigFromEnv() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return Config{
		DatabaseURL:       databaseURL,
		ConnectTimeout:    getDurationEnv("DB_CONNECT_TIMEOUT", defaultConnectTimeout),
		MaxConns:          getInt32Env("DB_MAX_CONNS", 10),
		MinConns:          getInt32Env("DB_MIN_CONNS", 1),
		MaxConnLifetime:   getDurationEnv("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime:   getDurationEnv("DB_MAX_CONN_IDLE_TIME", 15*time.Minute),
		HealthCheckPeriod: getDurationEnv("DB_HEALTH_CHECK_PERIOD", time.Minute),
	}, nil
}

// Open creates and returns a pgxpool.Pool configured from cfg. It does NOT
// verify connectivity — call Ping separately after Open to confirm the server
// is reachable. This separation allows callers to apply different error-handling
// strategies for "pool created but unreachable" versus "config is invalid".
func Open(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	return pool, nil
}

// Ping verifies that the pool can reach the Postgres server within timeout.
// It is intended to be called once at startup, after Open. The caller is
// responsible for closing the pool if Ping fails.
func Ping(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	return nil
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getInt32Env(key string, fallback int32) int32 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}

	return int32(parsed)
}
