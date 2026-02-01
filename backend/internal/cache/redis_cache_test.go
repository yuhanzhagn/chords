package cache

import (
	"context"
	"encoding/json"
//	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRedisClient mocks redis.Client methods used in RedisCache
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, ttl)
	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)
	return args.Get(0).(*redis.IntCmd)
}

// Helper to create a redis.StatusCmd that returns no error
func NewStatusCmdOK() *redis.StatusCmd {
	return redis.NewStatusCmd(context.Background())
}

// Helper to create a redis.StringCmd returning a value or error
func NewStringCmd(value string, err error) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetVal(value)
	if err != nil {
		cmd.SetErr(err)
	}
	return cmd
}

// Helper to create a redis.IntCmd returning a value or error
func NewIntCmd(val int64, err error) *redis.IntCmd {
	cmd := redis.NewIntCmd(context.Background())
	cmd.SetVal(val)
	if err != nil {
		cmd.SetErr(err)
	}
	return cmd
}

type MyStruct struct {
	Name string
	Age  int
}

func TestRedisCache_Mock(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockRedisClient)
	cache := &RedisCache[MyStruct]{client: mockClient, ctx: ctx}

	key := "user:1"
	val := MyStruct{Name: "Alice", Age: 30}

	// ---- Test Set ----
	data, _ := json.Marshal(val)
	mockClient.On("Set", ctx, key, data, time.Minute).Return(NewStatusCmdOK())

	err := cache.Set(key, val, time.Minute)
	require.NoError(t, err)
	mockClient.AssertCalled(t, "Set", ctx, key, data, time.Minute)

	// ---- Test Get (exists) ----
	jsonBytes, _ := json.Marshal(val)
	mockClient.On("Get", ctx, key).Return(NewStringCmd(string(jsonBytes), nil))

	got, ok := cache.Get(key)
	require.True(t, ok)
	require.Equal(t, val, got)

	// ---- Test Get (not exists) ----
	mockClient.On("Get", ctx, "missing").Return(NewStringCmd("", redis.Nil))

	got, ok = cache.Get("missing")
	require.False(t, ok)

	// ---- Test Delete ----
	mockClient.On("Del", ctx, []string{key}).Return(NewIntCmd(1, nil))

	err = cache.Delete(key)
	require.NoError(t, err)
}
