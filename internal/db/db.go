// Package db provides the database connection layer for api-go.
//
// It wraps jmoiron/sqlx to give callers three things:
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
//   - DB_MIN_CONNS=1     — keep at least one idle connection warm
//   - DB_MAX_CONN_LIFETIME=1h   — recycle connections to survive PgBouncer restarts
//   - DB_MAX_CONN_IDLE_TIME=15m — evict idle connections before server-side timeout
package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ErrNoRows is re-exported from database/sql so callers import the db package
// instead of database/sql directly for sentinel error checks.
// NOTE: db.ErrNoRows == sql.ErrNoRows (exact alias), so errors.Is checks are
// equivalent regardless of which sentinel a caller uses.
var ErrNoRows = sql.ErrNoRows

// DBTX is the minimal database execution interface shared by *sqlx.DB
// and *sqlx.Tx. Repositories accept DBTX rather than a concrete type so that:
//   - Normal calls run against the pool (connection auto-managed).
//   - Transactional calls pass a *sqlx.Tx without changing repository code.
//
// TRADE-OFF: This surface area is intentionally small (Exec, Query, QueryRow).
// Richer sqlx features (NamedExec, Preparex) require a type assertion.
type DBTX interface {
	sqlx.QueryerContext
	sqlx.ExecerContext
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
}

// WithTx runs fn inside a database transaction, committing on success and
// rolling back on any error. The deferred Rollback is a no-op after a
// successful Commit, making it safe to defer unconditionally.
//
// WHY: Centralizing transaction management here prevents callers from
// forgetting the rollback path, which historically causes connection leaks
// when a transaction is abandoned.
func WithTx(ctx context.Context, d *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // no-op after commit

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

const (
	defaultConnectTimeout = 10 * time.Second
)

// Config holds all tunable parameters for the Postgres connection pool.
// All fields except DatabaseURL have safe defaults applied by LoadConfigFromEnv,
// so callers do not need to set them explicitly in most environments.
type Config struct {
	DatabaseURL     string
	ConnectTimeout  time.Duration
	MaxOpenConns    int
	MaxIdleConns    int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
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
	databaseURL = normalizeDatabaseURL(databaseURL)

	return Config{
		DatabaseURL:     databaseURL,
		ConnectTimeout:  getDurationEnv("DB_CONNECT_TIMEOUT", defaultConnectTimeout),
		MaxOpenConns:    getIntEnv("DB_MAX_CONNS", 10),
		MaxIdleConns:    getIntEnv("DB_MIN_CONNS", 1),
		MaxConnLifetime: getDurationEnv("DB_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: getDurationEnv("DB_MAX_CONN_IDLE_TIME", 15*time.Minute),
	}, nil
}

// Open creates and returns a *sqlx.DB configured from cfg. It does NOT
// verify connectivity — call Ping separately after Open to confirm the server
// is reachable. This separation allows callers to apply different error-handling
// strategies for "pool created but unreachable" versus "config is invalid".
func Open(cfg Config) (*sqlx.DB, error) {
	d, err := sqlx.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	d.SetMaxOpenConns(cfg.MaxOpenConns)
	d.SetMaxIdleConns(cfg.MaxIdleConns)
	d.SetConnMaxLifetime(cfg.MaxConnLifetime)
	d.SetConnMaxIdleTime(cfg.MaxConnIdleTime)

	return d, nil
}

// Ping verifies that the database is reachable within timeout.
// It is intended to be called once at startup, after Open. The caller is
// responsible for closing the database if Ping fails.
func Ping(ctx context.Context, d *sqlx.DB, timeout time.Duration) error {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := d.PingContext(pingCtx); err != nil {
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

func getIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func normalizeDatabaseURL(raw string) string {
	if raw == "" {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.ReplaceAll(raw, "sslmode=prefer", "sslmode=disable")
	}

	query := parsed.Query()
	if strings.EqualFold(query.Get("sslmode"), "prefer") {
		query.Set("sslmode", "disable")
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}

	return raw
}
