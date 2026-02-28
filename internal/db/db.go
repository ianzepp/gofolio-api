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

// DBTX is the common interface satisfied by *pgxpool.Pool and pgx.Tx.
// Repositories accept this interface so the same code works inside or
// outside a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// WithTx runs fn inside a database transaction. If fn returns an error,
// the transaction is rolled back; otherwise it is committed.
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

type Config struct {
	DatabaseURL       string
	ConnectTimeout    time.Duration
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

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
