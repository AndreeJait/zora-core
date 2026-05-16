package outbound

import (
	"context"
	"fmt"

	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/go-utility/v2/no-sql/redisw"
	"github.com/redis/go-redis/v9"
)

// RedisConn wraps a Redis client with its cleanup function.
type RedisConn struct {
	Client *redis.Client
}

// ConnectRedis establishes a Redis connection.
// Returns a RedisConn and a cleanup function compatible with gracefulw.Register().
func ConnectRedis(ctx context.Context, cfg *config.AppConfig) (*RedisConn, func(ctx context.Context) error, error) {
	client, err := redisw.Connect(ctx, &redisw.Config{
		Address:      cfg.Redis.Address,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		DebugMode:    cfg.Redis.DebugMode,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	return &RedisConn{Client: client}, redisw.Disconnect(client), nil
}