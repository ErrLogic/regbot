package db

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// ConnectRedis creates a Redis client and verifies the connection.
func ConnectRedis(ctx context.Context, addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return rdb, nil
}
