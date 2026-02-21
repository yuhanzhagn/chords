package sink

import (
	"context"
	"time"
)

type RetrySinkConfig struct {
	Attempts int
	Backoff  func(attempt int) time.Duration
}

// RetrySink retries sink writes with configurable backoff.
type RetrySink[T any] struct {
	sink     Sink[T]
	attempts int
	backoff  func(attempt int) time.Duration
}

func NewRetrySink[T any](base Sink[T], cfg RetrySinkConfig) *RetrySink[T] {
	attempts := cfg.Attempts
	if attempts <= 0 {
		attempts = 3
	}
	backoff := cfg.Backoff
	if backoff == nil {
		backoff = func(attempt int) time.Duration {
			if attempt < 1 {
				attempt = 1
			}
			return time.Duration(attempt) * 50 * time.Millisecond
		}
	}

	return &RetrySink[T]{
		sink:     base,
		attempts: attempts,
		backoff:  backoff,
	}
}

func (r *RetrySink[T]) Write(ctx context.Context, value T) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var lastErr error
	for attempt := 1; attempt <= r.attempts; attempt++ {
		if err := r.sink.Write(ctx, value); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt == r.attempts {
			break
		}

		wait := r.backoff(attempt)
		if wait <= 0 {
			continue
		}

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}
