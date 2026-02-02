package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTypedCache_SetGetDelete(t *testing.T) {
	cache := NewTypedCache[string](time.Minute, time.Minute)

	key := "foo"
	value := "bar"

	// Set
	err := cache.Set(key, value, time.Minute)
	require.NoError(t, err)

	// Get
	got, ok := cache.Get(key)
	require.True(t, ok)
	require.Equal(t, value, got)

	// Delete
	err = cache.Delete(key)
	require.NoError(t, err)

	_, ok = cache.Get(key)
	require.False(t, ok)
}

func TestTypedCache_ZeroValueWhenWrongType(t *testing.T) {
	cache := NewTypedCache[float64](time.Minute, time.Minute)

	// store a string, but cache expects float64
	// cache.Set("key", any("not-a-float").(float64), time.Minute) // will panic in real code, so let's skip this
	// safer: simulate wrong type using underlying gocache
	cache.c.Set("key", "not-a-float", time.Minute)

	val, ok := cache.Get("key")
	require.False(t, ok)
	require.Equal(t, float64(0), val)
}

func TestTypedCache_Expiration(t *testing.T) {
	cache := NewTypedCache[string](50*time.Millisecond, 10*time.Millisecond)

	cache.Set("key", "value", 50*time.Millisecond)
	time.Sleep(60 * time.Millisecond)

	_, ok := cache.Get("key")
	require.False(t, ok)
}
