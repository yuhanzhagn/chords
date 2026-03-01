package redisdb

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

func DefaultConfig() Config {
	return Config{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	}
}

// NewClient creates and validates a Redis client against the provided server.
func NewClient(ctx context.Context, cfg Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Printf("Redis connected successfully: %s db=%d", cfg.Addr, cfg.DB)
	return client, nil
}

func InitRedis(ctx context.Context) (*redis.Client, error) {
	return NewClient(ctx, DefaultConfig())
}

func ClearRedis(rdb *redis.Client) {
	ctx := context.Background()
	err := rdb.FlushDB(ctx).Err() // flushes the current database
	if err != nil {
		log.Fatal("failed to flush redis:", err)
	}
}
