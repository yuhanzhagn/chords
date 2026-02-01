package cache

import "time"

type Cache[T any] interface {
    Set(key string, value T, ttl time.Duration) error
    Get(key string) (T, bool)
    Delete(key string) error
}
// Base Cache interface
