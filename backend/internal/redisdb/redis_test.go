package redisdb

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitializationFailureReturnsNoClient(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NotPanics(t, func() {
		client, err := NewClient(ctx, Config{Addr: "127.0.0.1:1"})
		require.Error(t, err)
		require.Nil(t, client)
	})
}
