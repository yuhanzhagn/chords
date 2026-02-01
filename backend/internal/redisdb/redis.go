package redisdb

import (
    "context"
    "log"
    "github.com/redis/go-redis/v9"
)

/*
var (
    Client *redis.Client
    ctx    = context.Background()
)*/

func InitRedis(ctx context.Context) (*redis.Client, error) {
    Client := redis.NewClient(&redis.Options{
        Addr:     "redis:6379",
        Password: "", // redis password, leave empty if none
        DB:       0,  // default DB
    })

    // Test connection
    if err := Client.Ping(ctx).Err(); err != nil {
        return nil, err
    }

    log.Println("Redis connected successfully")
    return Client, nil
}

func ClearRedis(rdb *redis.Client) {
    ctx := context.Background()
    err := rdb.FlushDB(ctx).Err() // flushes the current database
    if err != nil {
        log.Fatal("failed to flush redis:", err)
    }
}
