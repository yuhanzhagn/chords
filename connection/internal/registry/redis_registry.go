package registry

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	RoomUsersPrefix   string
	RoomUsersSuffix   string
	UserGatewayPrefix string
	UserGatewaySuffix string
	RoomUsersTTL      time.Duration
	UserGatewayTTL    time.Duration
}

type RedisRegistry struct {
	client *redis.Client
	cfg    Config
}

func NewRedisRegistry(client *redis.Client, cfg Config) *RedisRegistry {
	return &RedisRegistry{client: client, cfg: cfg}
}

func (r *RedisRegistry) AddRoomUser(ctx context.Context, roomID, userID uint32) error {
	if r == nil || r.client == nil {
		return nil
	}
	key := r.roomUsersKey(roomID)
	if err := r.client.SAdd(ctx, key, userID).Err(); err != nil {
		return err
	}
	if ttl := r.cfg.RoomUsersTTL; ttl > 0 {
		return r.client.Expire(ctx, key, ttl).Err()
	}
	return nil
}

func (r *RedisRegistry) RemoveRoomUser(ctx context.Context, roomID, userID uint32) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.SRem(ctx, r.roomUsersKey(roomID), userID).Err()
}

func (r *RedisRegistry) SetUserGateway(ctx context.Context, userID uint32, addr string) error {
	if r == nil || r.client == nil {
		return nil
	}
	if addr == "" {
		return nil
	}
	expire := r.cfg.UserGatewayTTL
	return r.client.Set(ctx, r.userGatewayKey(userID), addr, expire).Err()
}

func (r *RedisRegistry) RemoveUserGateway(ctx context.Context, userID uint32) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Del(ctx, r.userGatewayKey(userID)).Err()
}

func (r *RedisRegistry) roomUsersKey(roomID uint32) string {
	return fmt.Sprintf("%s%d%s", r.cfg.RoomUsersPrefix, roomID, r.cfg.RoomUsersSuffix)
}

func (r *RedisRegistry) userGatewayKey(userID uint32) string {
	return fmt.Sprintf("%s%d%s", r.cfg.UserGatewayPrefix, userID, r.cfg.UserGatewaySuffix)
}
