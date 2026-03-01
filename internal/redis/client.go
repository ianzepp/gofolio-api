package redis

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Client wraps go-redis so callers can depend on a small local type.
type Client struct {
	rdb *goredis.Client
}

// NewFromEnv creates a Redis client from REDIS_URL or REDIS_HOST/PORT.
// If neither is set, it returns (nil, nil) so the server can run without Redis.
func NewFromEnv(ctx context.Context) (*Client, error) {
	var opts *goredis.Options

	if rawURL := os.Getenv("REDIS_URL"); rawURL != "" {
		parsed, err := goredis.ParseURL(rawURL)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_URL: %w", err)
		}
		opts = parsed
	} else if host := os.Getenv("REDIS_HOST"); host != "" {
		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379"
		}
		db := 0
		if rawDB := os.Getenv("REDIS_DB"); rawDB != "" {
			parsed, err := strconv.Atoi(rawDB)
			if err != nil {
				return nil, fmt.Errorf("parse REDIS_DB: %w", err)
			}
			db = parsed
		}
		opts = &goredis.Options{
			Addr:     host + ":" + port,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       db,
		}
	} else {
		return nil, nil
	}

	rdb := goredis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

func (c *Client) Raw() *goredis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}
