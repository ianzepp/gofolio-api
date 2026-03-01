package cache

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

// Service provides cache operations backed by Redis.
type Service struct {
	rdb *goredis.Client
}

func NewService(rdb *goredis.Client) *Service {
	return &Service{rdb: rdb}
}

// Flush clears known Ghostfolio cache namespaces while leaving queue keys intact.
func (s *Service) Flush(ctx context.Context) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	patterns := []string{
		"quote-*",
		"portfolio-snapshot-*",
		"BENCHMARKS*",
	}
	for _, pattern := range patterns {
		if err := deleteByPattern(ctx, s.rdb, pattern); err != nil {
			return err
		}
	}
	return nil
}

func deleteByPattern(ctx context.Context, rdb *goredis.Client, pattern string) error {
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, pattern, 500).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
