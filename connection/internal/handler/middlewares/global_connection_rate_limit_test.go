package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGlobalConnectionRateLimitMiddleware_AllowsRequestsWithinBurst(t *testing.T) {
	now := time.Unix(1000, 0)
	mw := GlobalConnectionRateLimitMiddleware(GlobalConnectionRateLimitOptions{
		RatePerSecond: 1,
		Burst:         2,
		Now: func() time.Time {
			return now
		},
	})

	hit := 0
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit++
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("request %d should pass, got status %d", i+1, rr.Code)
		}
	}

	if hit != 2 {
		t.Fatalf("next handler should be called twice, got %d", hit)
	}
}

func TestGlobalConnectionRateLimitMiddleware_RejectsWhenBurstExceeded(t *testing.T) {
	now := time.Unix(2000, 0)
	mw := GlobalConnectionRateLimitMiddleware(GlobalConnectionRateLimitOptions{
		RatePerSecond: 1,
		Burst:         1,
		Now: func() time.Time {
			return now
		},
	})

	hit := 0
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit++
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/ws", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request should pass, got status %d", first.Code)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/ws", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request should be limited, got status %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("limited response should include Retry-After")
	}

	if hit != 1 {
		t.Fatalf("next handler should be called once, got %d", hit)
	}
}

func TestGlobalConnectionRateLimitMiddleware_RefillsOverTime(t *testing.T) {
	base := time.Unix(3000, 0)
	now := base
	mw := GlobalConnectionRateLimitMiddleware(GlobalConnectionRateLimitOptions{
		RatePerSecond: 1,
		Burst:         1,
		Now: func() time.Time {
			return now
		},
	})

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/ws", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request should pass, got status %d", first.Code)
	}

	second := httptest.NewRecorder()
	h.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/ws", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second immediate request should be limited, got status %d", second.Code)
	}

	now = now.Add(1200 * time.Millisecond)
	third := httptest.NewRecorder()
	h.ServeHTTP(third, httptest.NewRequest(http.MethodGet, "/ws", nil))
	if third.Code != http.StatusNoContent {
		t.Fatalf("request after refill should pass, got status %d", third.Code)
	}
}
