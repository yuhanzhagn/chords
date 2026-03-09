package middlewares

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// GlobalConnectionRateLimitOptions configures GlobalConnectionRateLimitMiddleware.
type GlobalConnectionRateLimitOptions struct {
	// RatePerSecond is the global token refill rate.
	RatePerSecond float64
	// Burst is the max number of upgrade attempts that can happen at once.
	Burst int
	// Now provides current time. Defaults to time.Now.
	Now func() time.Time
	// OnLimit is called when request is rejected. Defaults to writing 429 + Retry-After.
	OnLimit func(http.ResponseWriter, *http.Request, time.Duration)
}

type globalBucket struct {
	tokens     float64
	lastRefill time.Time
}

// GlobalConnectionRateLimitMiddleware limits websocket handshake requests globally.
//
// This is useful for smoothing reconnect storms where many clients retry at once.
func GlobalConnectionRateLimitMiddleware(opts GlobalConnectionRateLimitOptions) func(http.Handler) http.Handler {
	rate := opts.RatePerSecond
	if rate <= 0 {
		rate = 30
	}

	burst := opts.Burst
	if burst <= 0 {
		burst = 60
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	onLimit := opts.OnLimit
	if onLimit == nil {
		onLimit = defaultGlobalRateLimitResponder
	}

	var mu sync.Mutex
	bucket := globalBucket{tokens: float64(burst), lastRefill: nowFn()}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := nowFn()

			mu.Lock()
			refillGlobalBucket(&bucket, now, rate, burst)
			if bucket.tokens < 1 {
				wait := time.Duration(math.Ceil((1-bucket.tokens)/rate*float64(time.Second)))
				mu.Unlock()
				onLimit(w, r, wait)
				return
			}

			bucket.tokens -= 1
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func refillGlobalBucket(bucket *globalBucket, now time.Time, rate float64, burst int) {
	if bucket == nil {
		return
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(float64(burst), bucket.tokens+elapsed*rate)
		bucket.lastRefill = now
	}
}

func defaultGlobalRateLimitResponder(w http.ResponseWriter, _ *http.Request, wait time.Duration) {
	if wait < 0 {
		wait = 0
	}
	retryAfter := int(math.Ceil(wait.Seconds()))
	if retryAfter < 1 {
		retryAfter = 1
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	http.Error(w, "too many connection attempts", http.StatusTooManyRequests)
}
