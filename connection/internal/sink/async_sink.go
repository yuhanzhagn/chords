package sink

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrAsyncSinkClosed = errors.New("async sink is closed")
	ErrAsyncBufferFull = errors.New("async sink buffer is full")
)

type AsyncSinkConfig struct {
	BufferSize int
	Workers    int
	// BlockOnEnqueue controls write behavior when the buffer is full.
	// false keeps the gateway loop responsive and returns ErrAsyncBufferFull.
	BlockOnEnqueue bool
	OnWriteError   func(error)
}

type closeable interface {
	Close() error
}

type asyncJob[T any] struct {
	ctx   context.Context
	value T
}

// AsyncSink decouples producers from sink writes with a buffered queue.
type AsyncSink[T any] struct {
	sink           Sink[T]
	jobs           chan asyncJob[T]
	blockOnEnqueue bool
	onWriteError   func(error)

	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
}

func NewAsyncSink[T any](base Sink[T], cfg AsyncSinkConfig) *AsyncSink[T] {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1024
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}

	s := &AsyncSink[T]{
		sink:           base,
		jobs:           make(chan asyncJob[T], cfg.BufferSize),
		blockOnEnqueue: cfg.BlockOnEnqueue,
		onWriteError:   cfg.OnWriteError,
	}
	s.wg.Add(cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		go s.worker()
	}
	return s
}

func (s *AsyncSink[T]) Write(ctx context.Context, value T) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrAsyncSinkClosed
	}

	job := asyncJob[T]{ctx: ctx, value: value}

	if s.blockOnEnqueue {
		select {
		case s.jobs <- job:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	select {
	case s.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrAsyncBufferFull
	}
}

func (s *AsyncSink[T]) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.jobs)
	s.mu.Unlock()

	s.wg.Wait()
	if c, ok := s.sink.(closeable); ok {
		if err := c.Close(); err != nil {
			return fmt.Errorf("close underlying sink: %w", err)
		}
	}
	return nil
}

func (s *AsyncSink[T]) worker() {
	defer s.wg.Done()
	for job := range s.jobs {
		if err := s.sink.Write(job.ctx, job.value); err != nil && s.onWriteError != nil {
			s.onWriteError(err)
		}
	}
}
