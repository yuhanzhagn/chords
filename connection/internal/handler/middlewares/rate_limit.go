package middlewares

import (
	"connection/internal/handler"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"
)

type connectionBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

// ConnectionRateLimitOptions configures ConnectionRateLimitMiddleware.
type ConnectionRateLimitOptions struct {
	// RatePerSecond is the token refill rate for each connection.
	RatePerSecond float64
	// Burst is the max number of tokens a connection can accumulate.
	Burst int
	// IdleTTL controls when inactive connection buckets are evicted.
	IdleTTL time.Duration
	// Now provides current time. Defaults to time.Now.
	Now func() time.Time
}

// ConnectionRateLimitMiddleware limits inbound event handling rate per websocket connection.
func ConnectionRateLimitMiddleware(opts ConnectionRateLimitOptions) handler.Middleware {
	rate := opts.RatePerSecond
	if rate <= 0 {
		rate = 10
	}

	burst := opts.Burst
	if burst <= 0 {
		burst = 20
	}

	idleTTL := opts.IdleTTL
	if idleTTL <= 0 {
		idleTTL = 5 * time.Minute
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	var mu sync.Mutex
	buckets := make(map[string]*connectionBucket)
	calls := 0

	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(c *handler.Context) error {
			if c == nil {
				return errors.New("context is required")
			}

			req, err := requestFromContext(c)
			if err != nil {
				return err
			}

			now := nowFn()
			key := connectionKey(req)

			mu.Lock()
			calls++
			if calls%128 == 0 {
				evictIdleBuckets(buckets, now, idleTTL)
			}

			bucket, ok := buckets[key]
			if !ok {
				bucket = &connectionBucket{
					tokens:     float64(burst),
					lastRefill: now,
					lastSeen:   now,
				}
				buckets[key] = bucket
			}

			refillBucket(bucket, now, rate, burst)
			if bucket.tokens < 1 {
				bucket.lastSeen = now
				mu.Unlock()
				return errors.New("rate limit exceeded for connection")
			}

			bucket.tokens -= 1
			bucket.lastSeen = now
			mu.Unlock()

			return next(c)
		}
	}
}

func refillBucket(bucket *connectionBucket, now time.Time, rate float64, burst int) {
	if bucket == nil {
		return
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(float64(burst), bucket.tokens+elapsed*rate)
		bucket.lastRefill = now
	}
}

func evictIdleBuckets(buckets map[string]*connectionBucket, now time.Time, idleTTL time.Duration) {
	for key, bucket := range buckets {
		if bucket == nil {
			delete(buckets, key)
			continue
		}
		if now.Sub(bucket.lastSeen) >= idleTTL {
			delete(buckets, key)
		}
	}
}

func connectionKey(req *http.Request) string {
	return fmt.Sprintf("%p", req)
}
