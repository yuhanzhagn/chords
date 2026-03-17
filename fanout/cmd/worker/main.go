package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"fanout/internal/app"
	"fanout/internal/fanout"
	"fanout/internal/kafka"
	"fanout/internal/registry"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := mustLoadConfig("configs/config.yaml")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	reg := registry.NewRedisRegistry(redisClient, registry.Config{
		RoomUsersPrefix:   cfg.Redis.RoomUsersPrefix,
		RoomUsersSuffix:   cfg.Redis.RoomUsersSuffix,
		UserGatewayPrefix: cfg.Redis.UserGatewayPrefix,
		UserGatewaySuffix: cfg.Redis.UserGatewaySuffix,
	})

	httpClient := &http.Client{Timeout: cfg.Fanout.RequestTimeout}
	dispatcher := fanout.NewDispatcher(reg, httpClient, fanout.Config{GatewayPath: cfg.Fanout.GatewayPath})

	consumer, err := kafka.NewNotificationConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.ConsumerGroup,
		cfg.Kafka.Topics,
		dispatcher.Dispatch,
	)
	if err != nil {
		log.Fatalf("failed to create kafka consumer: %v", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("failed to close consumer: %v", err)
		}
	}()

	consumer.Start(ctx)
	<-ctx.Done()

	if err := redisClient.Close(); err != nil {
		log.Printf("redis close failed: %v", err)
	}
}

func mustLoadConfig(path string) *app.Config {
	cfg, err := app.LoadConfig(path)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	return cfg
}
