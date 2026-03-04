package middlewares

import (
	"connection/internal/handler"
	"errors"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unsafe"
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
	var evictionTimer *time.Timer
	var evictionTimerC <-chan time.Time

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
			if evictionTimerC != nil {
				select {
				case <-evictionTimerC:
					// Timer-driven eviction keeps request path fast under high connection counts.
					evictIdleBuckets(buckets, now, idleTTL)
					if len(buckets) == 0 {
						stopEvictionTimer(&evictionTimer, &evictionTimerC)
					} else {
						resetEvictionTimer(&evictionTimer, &evictionTimerC, nextEvictionDelay(buckets, now, idleTTL))
					}
				default:
				}
			}

			bucket, ok := buckets[key]
			if !ok {
				bucket = &connectionBucket{
					tokens:     float64(burst),
					lastRefill: now,
					lastSeen:   now,
				}
				buckets[key] = bucket
				if evictionTimer == nil {
					resetEvictionTimer(&evictionTimer, &evictionTimerC, idleTTL)
				}
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

func nextEvictionDelay(buckets map[string]*connectionBucket, now time.Time, idleTTL time.Duration) time.Duration {
	if len(buckets) == 0 {
		return idleTTL
	}

	soonest := idleTTL
	for _, bucket := range buckets {
		if bucket == nil {
			return 0
		}

		wait := bucket.lastSeen.Add(idleTTL).Sub(now)
		if wait <= 0 {
			return 0
		}
		if wait < soonest {
			soonest = wait
		}
	}
	return soonest
}

func stopEvictionTimer(timer **time.Timer, timerC *<-chan time.Time) {
	if *timer == nil {
		*timerC = nil
		return
	}

	if !(*timer).Stop() {
		select {
		case <-(*timer).C:
		default:
		}
	}
	*timer = nil
	*timerC = nil
}

func resetEvictionTimer(timer **time.Timer, timerC *<-chan time.Time, wait time.Duration) {
	if wait < 0 {
		wait = 0
	}

	if *timer == nil {
		t := time.NewTimer(wait)
		*timer = t
		*timerC = t.C
		return
	}

	if !(*timer).Stop() {
		select {
		case <-(*timer).C:
		default:
		}
	}
	(*timer).Reset(wait)
	*timerC = (*timer).C
}

func connectionKey(req *http.Request) string {
	if req == nil {
		return ""
	}

	// RFC 6455: client-generated key per websocket handshake; stable for a connection.
	if wsKey := req.Header.Get("Sec-WebSocket-Key"); wsKey != "" {
		return wsKey
	}

	if req.RemoteAddr != "" {
		return req.RemoteAddr
	}

	if req.Host != "" {
		return req.Host
	}

	if req.RequestURI != "" {
		return req.RequestURI
	}

	return strconv.FormatUint(uint64(uintptr(unsafe.Pointer(req))), 36)
}
