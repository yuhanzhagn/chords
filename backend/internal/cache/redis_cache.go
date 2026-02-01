package cache

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

type RedisClient interface {
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd
    Get(ctx context.Context, key string) *redis.StringCmd
    Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type RedisCache[T any] struct {
    client RedisClient
    ctx    context.Context
}

func NewRedisCache[T any](client *redis.Client) *RedisCache[T] {
    return &RedisCache[T]{
        client: client,
        ctx:    context.Background(),
    }
}

func (r *RedisCache[T]) Set(key string, value T, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return r.client.Set(r.ctx, key, data, ttl).Err()
}

func (r *RedisCache[T]) Get(key string) (T, bool) {
    var result T
    data, err := r.client.Get(r.ctx, key).Bytes()
    if err != nil {
        if err == redis.Nil {
            return result, false
        }
        return result, false
    }
    if err := json.Unmarshal(data, &result); err != nil {
        return result, false
    }
    return result, true
}

func (r *RedisCache[T]) Delete(key string) error {
    return r.client.Del(r.ctx, key).Err()
}

