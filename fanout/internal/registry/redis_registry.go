package registry

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

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

func (r *RedisRegistry) RoomUsers(ctx context.Context, roomID uint32) ([]uint32, error) {
	key := r.roomUsersKey(roomID)
	members, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers %s: %w", key, err)
	}

	userIDs := make([]uint32, 0, len(members))
	for _, member := range members {
		val := strings.TrimSpace(member)
		if val == "" {
			continue
		}
		parsed, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			log.Printf("[fanout-registry] invalid user id %q in %s", val, key)
			continue
		}
		userIDs = append(userIDs, uint32(parsed))
	}

	return userIDs, nil
}

func (r *RedisRegistry) UserGateways(ctx context.Context, userIDs []uint32) (map[uint32]string, error) {
	if len(userIDs) == 0 {
		return map[uint32]string{}, nil
	}

	keys := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		keys = append(keys, r.userGatewayKey(id))
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis mget: %w", err)
	}

	result := make(map[uint32]string, len(values))
	for i, val := range values {
		if val == nil {
			continue
		}
		var addr string
		switch typed := val.(type) {
		case string:
			addr = typed
		case []byte:
			addr = string(typed)
		default:
			addr = fmt.Sprint(val)
		}
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		result[userIDs[i]] = addr
	}
	return result, nil
}

func (r *RedisRegistry) roomUsersKey(roomID uint32) string {
	return fmt.Sprintf("%s%d%s", r.cfg.RoomUsersPrefix, roomID, r.cfg.RoomUsersSuffix)
}

func (r *RedisRegistry) userGatewayKey(userID uint32) string {
	return fmt.Sprintf("%s%d%s", r.cfg.UserGatewayPrefix, userID, r.cfg.UserGatewaySuffix)
}
