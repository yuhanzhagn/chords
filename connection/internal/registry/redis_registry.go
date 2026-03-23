package registry

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	RoomUsersPrefix   string
	RoomUsersSuffix   string
	UserGatewayPrefix string
	UserGatewaySuffix string
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
	return r.client.SAdd(ctx, r.roomUsersKey(roomID), userID).Err()
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
	return r.client.Set(ctx, r.userGatewayKey(userID), addr, 0).Err()
}

func (r *RedisRegistry) roomUsersKey(roomID uint32) string {
	return fmt.Sprintf("%s%d%s", r.cfg.RoomUsersPrefix, roomID, r.cfg.RoomUsersSuffix)
}

func (r *RedisRegistry) userGatewayKey(userID uint32) string {
	return fmt.Sprintf("%s%d%s", r.cfg.UserGatewayPrefix, userID, r.cfg.UserGatewaySuffix)
}
