package cache

import (
	gocache "github.com/patrickmn/go-cache"
	"time"
)

type TypedCache[T any] struct {
	c *gocache.Cache
}

func NewTypedCache[T any](defaultExp, cleanupInterval time.Duration) *TypedCache[T] {
	return &TypedCache[T]{
		c: gocache.New(defaultExp, cleanupInterval),
	}
}

func (g *TypedCache[T]) Set(key string, value T, ttl time.Duration) error {
	g.c.Set(key, value, ttl)
	return nil
}

func (g *TypedCache[T]) Get(key string) (T, bool) {
	v, ok := g.c.Get(key)
	if !ok {
		var zero T
		return zero, false
	}

	val, ok := v.(T)
	if !ok {
		var zero T
		return zero, false
	}

	return val, true
}

func (g *TypedCache[T]) Delete(key string) error {
	g.c.Delete(key)
	return nil
}
