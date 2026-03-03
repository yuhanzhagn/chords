package middlewares

import (
	"connection/internal/handler"
	"context"
	"net/http"
	"testing"
	"time"
)

func TestConnectionRateLimitMiddleware_SameConnectionIsLimited(t *testing.T) {
	now := time.Unix(1000, 0)
	mw := ConnectionRateLimitMiddleware(ConnectionRateLimitOptions{
		RatePerSecond: 1,
		Burst:         2,
		Now: func() time.Time {
			return now
		},
	})

	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	ctx := &handler.Context{
		Context: context.Background(),
		Values: map[string]any{
			handler.RequestContextKey: req,
		},
	}

	h := mw(func(_ *handler.Context) error { return nil })

	if err := h(ctx); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}
	if err := h(ctx); err != nil {
		t.Fatalf("second request should pass: %v", err)
	}
	if err := h(ctx); err == nil {
		t.Fatal("third request should be rate limited")
	}
}

func TestConnectionRateLimitMiddleware_DifferentConnectionsAreIsolated(t *testing.T) {
	now := time.Unix(2000, 0)
	mw := ConnectionRateLimitMiddleware(ConnectionRateLimitOptions{
		RatePerSecond: 1,
		Burst:         1,
		Now: func() time.Time {
			return now
		},
	})

	reqA, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	reqB, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	ctxA := &handler.Context{
		Context: context.Background(),
		Values: map[string]any{
			handler.RequestContextKey: reqA,
		},
	}
	ctxB := &handler.Context{
		Context: context.Background(),
		Values: map[string]any{
			handler.RequestContextKey: reqB,
		},
	}

	h := mw(func(_ *handler.Context) error { return nil })

	if err := h(ctxA); err != nil {
		t.Fatalf("connection A should pass first event: %v", err)
	}
	if err := h(ctxA); err == nil {
		t.Fatal("connection A should be rate limited on second event")
	}
	if err := h(ctxB); err != nil {
		t.Fatalf("connection B should still pass first event: %v", err)
	}
}

func TestConnectionRateLimitMiddleware_RefillsAfterTime(t *testing.T) {
	base := time.Unix(3000, 0)
	now := base
	mw := ConnectionRateLimitMiddleware(ConnectionRateLimitOptions{
		RatePerSecond: 1,
		Burst:         1,
		Now: func() time.Time {
			return now
		},
	})

	req, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	ctx := &handler.Context{
		Context: context.Background(),
		Values: map[string]any{
			handler.RequestContextKey: req,
		},
	}

	h := mw(func(_ *handler.Context) error { return nil })

	if err := h(ctx); err != nil {
		t.Fatalf("first request should pass: %v", err)
	}
	if err := h(ctx); err == nil {
		t.Fatal("second immediate request should be rate limited")
	}

	now = now.Add(1200 * time.Millisecond)
	if err := h(ctx); err != nil {
		t.Fatalf("request after refill should pass: %v", err)
	}
}
