package source

import (
	"connection/internal/handler"
	"context"
	"errors"
	"sync"
)

// Source defines a lifecycle contract for inbound message producers.
// Implementations should start background consumption in Start and stop gracefully in Stop.
type Source[T any] interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// BaseSource provides shared lifecycle and dependency scaffolding for concrete sources.
// It centralizes cancellation and goroutine coordination while remaining transport-agnostic.
type BaseSource[T any] struct {
	handler handler.HandlerFunc

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	started bool
}

// NewBaseSource constructs common source scaffolding with constructor-injected handler.
func NewBaseSource[T any](h handler.HandlerFunc) (*BaseSource[T], error) {
	if h == nil {
		return nil, errors.New("handler is required")
	}

	return &BaseSource[T]{
		handler: h,
	}, nil
}

// Handler returns the injected message handler.
func (b *BaseSource[T]) Handler() handler.HandlerFunc {
	return b.handler
}

// StartLoop runs a non-blocking background loop controlled by context cancellation.
func (b *BaseSource[T]) StartLoop(ctx context.Context, run func(ctx context.Context)) error {
	if run == nil {
		return errors.New("run function is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return errors.New("source already started")
	}

	workerCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	b.done = make(chan struct{})
	b.started = true

	go func() {
		defer close(b.done)
		run(workerCtx)
	}()

	return nil
}

// Stop requests background shutdown and waits for completion or caller timeout.
func (b *BaseSource[T]) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return nil
	}

	cancel := b.cancel
	done := b.done
	b.cancel = nil
	b.done = nil
	b.started = false
	b.mu.Unlock()

	cancel()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
