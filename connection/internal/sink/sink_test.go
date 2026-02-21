package sink

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type stubSink[T any] struct {
	writeFn func(context.Context, T) error
	closeFn func() error
}

func (s *stubSink[T]) Write(ctx context.Context, value T) error {
	if s.writeFn != nil {
		return s.writeFn(ctx, value)
	}
	return nil
}

func (s *stubSink[T]) Close() error {
	if s.closeFn != nil {
		return s.closeFn()
	}
	return nil
}

func TestNewMultiError(t *testing.T) {
	e1 := errors.New("one")
	e2 := errors.New("two")

	if err := NewMultiError(nil, nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if err := NewMultiError(e1); !errors.Is(err, e1) {
		t.Fatalf("expected single error passthrough")
	}

	err := NewMultiError(e1, nil, e2)
	var me *MultiError
	if !errors.As(err, &me) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(me.Errors()) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(me.Errors()))
	}
	if !errors.Is(me, e1) || !errors.Is(me, e2) {
		t.Fatalf("expected errors.Is to match all underlying errors")
	}
}

func TestMultiSinkSequential(t *testing.T) {
	e1 := errors.New("sink1")
	e2 := errors.New("sink2")
	ms := NewMultiSink[int](MultiSinkConfig{Concurrent: false},
		&stubSink[int]{writeFn: func(context.Context, int) error { return e1 }},
		&stubSink[int]{writeFn: func(context.Context, int) error { return nil }},
		&stubSink[int]{writeFn: func(context.Context, int) error { return e2 }},
	)

	err := ms.Write(context.Background(), 1)
	var me *MultiError
	if !errors.As(err, &me) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(me.Errors()) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(me.Errors()))
	}
	if !errors.Is(me, e1) || !errors.Is(me, e2) {
		t.Fatalf("expected wrapped errors to be preserved")
	}
}

func TestMultiSinkConcurrent(t *testing.T) {
	var count atomic.Int32
	ms := NewMultiSink[int](MultiSinkConfig{Concurrent: true},
		&stubSink[int]{writeFn: func(context.Context, int) error {
			time.Sleep(15 * time.Millisecond)
			count.Add(1)
			return nil
		}},
		&stubSink[int]{writeFn: func(context.Context, int) error {
			time.Sleep(15 * time.Millisecond)
			count.Add(1)
			return nil
		}},
	)

	start := time.Now()
	if err := ms.Write(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	if count.Load() != 2 {
		t.Fatalf("expected 2 writes, got %d", count.Load())
	}
	if elapsed >= 30*time.Millisecond {
		t.Fatalf("expected concurrent execution, elapsed %v", elapsed)
	}
}

func TestAsyncSinkNonBlocking(t *testing.T) {
	block := make(chan struct{})
	base := &stubSink[int]{writeFn: func(context.Context, int) error {
		<-block
		return nil
	}}
	s := NewAsyncSink[int](base, AsyncSinkConfig{
		BufferSize:     1,
		Workers:        1,
		BlockOnEnqueue: false,
		OnWriteError:   nil,
	})

	if err := s.Write(context.Background(), 1); err != nil {
		t.Fatalf("first enqueue failed: %v", err)
	}
	if err := s.Write(context.Background(), 2); err == nil {
		t.Fatalf("expected buffer full error")
	} else if !errors.Is(err, ErrAsyncBufferFull) {
		t.Fatalf("expected ErrAsyncBufferFull, got %v", err)
	}

	close(block)
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestAsyncSinkCloseWaitsAndClosesUnderlying(t *testing.T) {
	var writes atomic.Int32
	closed := make(chan struct{}, 1)
	base := &stubSink[int]{
		writeFn: func(context.Context, int) error {
			time.Sleep(10 * time.Millisecond)
			writes.Add(1)
			return nil
		},
		closeFn: func() error {
			closed <- struct{}{}
			return nil
		},
	}
	s := NewAsyncSink[int](base, AsyncSinkConfig{
		BufferSize:     8,
		Workers:        2,
		BlockOnEnqueue: true,
	})
	for i := 0; i < 4; i++ {
		if err := s.Write(context.Background(), i); err != nil {
			t.Fatalf("enqueue %d failed: %v", i, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if writes.Load() != 4 {
		t.Fatalf("expected 4 writes before close, got %d", writes.Load())
	}
	select {
	case <-closed:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected underlying close to be called")
	}
	if err := s.Write(context.Background(), 5); !errors.Is(err, ErrAsyncSinkClosed) {
		t.Fatalf("expected ErrAsyncSinkClosed, got %v", err)
	}
}

func TestRetrySink(t *testing.T) {
	var attempts atomic.Int32
	retryableErr := errors.New("temporary")
	base := &stubSink[int]{
		writeFn: func(context.Context, int) error {
			n := attempts.Add(1)
			if n < 3 {
				return retryableErr
			}
			return nil
		},
	}
	rs := NewRetrySink[int](base, RetrySinkConfig{
		Attempts: 5,
		Backoff: func(int) time.Duration {
			return 0
		},
	})

	if err := rs.Write(context.Background(), 123); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetrySinkContextCancel(t *testing.T) {
	base := &stubSink[int]{
		writeFn: func(context.Context, int) error {
			return errors.New("always fail")
		},
	}
	rs := NewRetrySink[int](base, RetrySinkConfig{
		Attempts: 5,
		Backoff: func(int) time.Duration {
			return 50 * time.Millisecond
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := rs.Write(ctx, 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestAsyncSinkErrorCallback(t *testing.T) {
	base := &stubSink[int]{
		writeFn: func(context.Context, int) error {
			return errors.New("write failure")
		},
	}
	var mu sync.Mutex
	got := 0
	s := NewAsyncSink[int](base, AsyncSinkConfig{
		BufferSize:     4,
		Workers:        1,
		BlockOnEnqueue: true,
		OnWriteError: func(err error) {
			if err == nil {
				return
			}
			mu.Lock()
			got++
			mu.Unlock()
		},
	})

	if err := s.Write(context.Background(), 1); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got != 1 {
		t.Fatalf("expected callback count 1, got %d", got)
	}
}
