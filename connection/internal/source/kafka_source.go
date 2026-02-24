package source

import (
	"connection/internal/handler"
	"context"
	"time"
)

// KafkaSourceOptions configures the scaffold behavior for KafkaSource.
// The defaults are safe for stub wiring in dependency injection.
type KafkaSourceOptions[T any] struct {
	Topic          string
	PollInterval   time.Duration
	MessageFactory func() T
	OnHandleError  func(error)
}

// KafkaSource is a stubbed inbound source that simulates Kafka consumption.
// It is intentionally transport-light and only forwards messages to Handler.
type KafkaSource[T any] struct {
	base *BaseSource[T]

	topic          string
	pollInterval   time.Duration
	messageFactory func() T
	onHandleError  func(error)
}

// NewKafkaSource creates a handler-injected source that can be wired from main.go.
func NewKafkaSource[T any](h handler.HandlerFunc, opts KafkaSourceOptions[T]) (*KafkaSource[T], error) {
	base, err := NewBaseSource[T](h)
	if err != nil {
		return nil, err
	}

	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}

	messageFactory := opts.MessageFactory
	if messageFactory == nil {
		messageFactory = func() T {
			var zero T
			return zero
		}
	}

	return &KafkaSource[T]{
		base:           base,
		topic:          opts.Topic,
		pollInterval:   pollInterval,
		messageFactory: messageFactory,
		onHandleError:  opts.OnHandleError,
	}, nil
}

// Start launches a background consumption loop and returns immediately.
func (s *KafkaSource[T]) Start(ctx context.Context) error {
	return s.base.StartLoop(ctx, s.consumeLoop)
}

// Stop cancels the loop and waits for graceful shutdown.
func (s *KafkaSource[T]) Stop(ctx context.Context) error {
	return s.base.Stop(ctx)
}

func (s *KafkaSource[T]) consumeLoop(ctx context.Context) {
	_ = s.topic // retained for future real Kafka client wiring.

	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			message := s.messageFactory()
			eventCtx := &handler.Context{
				Context:    ctx,
				Event:      message,
				ReceivedAt: time.Now(),
			}
			if err := s.base.Handler()(eventCtx); err != nil && s.onHandleError != nil {
				s.onHandleError(err)
			}
		}
	}
}
