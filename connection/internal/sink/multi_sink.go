package sink

import (
	"context"
	"fmt"
	"sync"
)

type MultiSinkConfig struct {
	Concurrent bool
}

// MultiSink fans out each value to multiple sinks.
type MultiSink[T any] struct {
	sinks      []Sink[T]
	concurrent bool
}

func NewMultiSink[T any](cfg MultiSinkConfig, sinks ...Sink[T]) *MultiSink[T] {
	filtered := make([]Sink[T], 0, len(sinks))
	for _, s := range sinks {
		if s != nil {
			filtered = append(filtered, s)
		}
	}
	return &MultiSink[T]{
		sinks:      filtered,
		concurrent: cfg.Concurrent,
	}
}

func (m *MultiSink[T]) Write(ctx context.Context, value T) error {
	if len(m.sinks) == 0 {
		return nil
	}

	if !m.concurrent {
		errs := make([]error, 0, len(m.sinks))
		for i, s := range m.sinks {
			if err := s.Write(ctx, value); err != nil {
				errs = append(errs, fmt.Errorf("sink[%d]: %w", i, err))
			}
		}
		return NewMultiError(errs...)
	}

	errCh := make(chan error, len(m.sinks))
	var wg sync.WaitGroup
	wg.Add(len(m.sinks))
	for i, s := range m.sinks {
		go func(index int, sink Sink[T]) {
			defer wg.Done()
			if err := sink.Write(ctx, value); err != nil {
				errCh <- fmt.Errorf("sink[%d]: %w", index, err)
			}
		}(i, s)
	}
	wg.Wait()
	close(errCh)

	errs := make([]error, 0, len(m.sinks))
	for err := range errCh {
		errs = append(errs, err)
	}
	return NewMultiError(errs...)
}
