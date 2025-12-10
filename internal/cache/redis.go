package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Anvoria/authly/internal/config"
	"github.com/redis/go-redis/v9"
)

var (
	// RedisClient is the global Redis client instance
	RedisClient *redis.Client
)

// ConnectRedis initializes the package-level RedisClient and verifies connectivity to Redis.
// It creates a redis.Client from cfg (address, password, DB), performs a Ping using a 5-second timeout, and logs a success message.
// It returns an error if the initial connectivity test fails.
func ConnectRedis(cfg *config.RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	slog.Info("Redis connected successfully", "address", cfg.Address())
	return nil
}

// CloseRedis closes the global RedisClient if it is initialized.
// It returns any error from the underlying Close call and does nothing if RedisClient is nil.
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}