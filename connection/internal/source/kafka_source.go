package source

import (
	"connection/internal/event/codec"
	"connection/internal/handler"
	"context"
	"errors"
	"log"
	"time"

	"github.com/IBM/sarama"
)

// KafkaSourceOptions configures Kafka consumer behavior for KafkaSource.
type KafkaSourceOptions[T any] struct {
	Brokers        []string
	GroupID        string
	Topics         []string
	Decoder        codec.EventDecoder[T]
	SaramaConfig   *sarama.Config
	OnHandleError  func(error)
}

// KafkaSource consumes Kafka messages and forwards decoded events to Handler.
type KafkaSource[T any] struct {
	base *BaseSource[T]

	consumer      sarama.ConsumerGroup
	topics        []string
	decoder       codec.EventDecoder[T]
	onHandleError func(error)
}

// NewKafkaSource creates a handler-injected Kafka source.
func NewKafkaSource[T any](h handler.HandlerFunc, opts KafkaSourceOptions[T]) (*KafkaSource[T], error) {
	base, err := NewBaseSource[T](h)
	if err != nil {
		return nil, err
	}
	if len(opts.Brokers) == 0 {
		return nil, errors.New("brokers are required")
	}
	if opts.GroupID == "" {
		return nil, errors.New("group id is required")
	}
	if len(opts.Topics) == 0 {
		return nil, errors.New("topics are required")
	}
	if opts.Decoder == nil {
		return nil, errors.New("decoder is required")
	}

	cfg := opts.SaramaConfig
	if cfg == nil {
		cfg = sarama.NewConfig()
		cfg.Version = sarama.V2_8_0_0
		cfg.Consumer.Return.Errors = true
		cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	cg, err := sarama.NewConsumerGroup(opts.Brokers, opts.GroupID, cfg)
	if err != nil {
		return nil, err
	}

	return &KafkaSource[T]{
		base:          base,
		consumer:      cg,
		topics:        opts.Topics,
		decoder:       opts.Decoder,
		onHandleError: opts.OnHandleError,
	}, nil
}

type consumerGroupHandler[T any] struct {
	source *KafkaSource[T]
	ctx    context.Context
}

func (h *consumerGroupHandler[T]) Setup(_ sarama.ConsumerGroupSession) error {
	log.Println("[kafka-source] setup")
	return nil
}

func (h *consumerGroupHandler[T]) Cleanup(_ sarama.ConsumerGroupSession) error {
	log.Println("[kafka-source] cleanup")
	return nil
}

func (h *consumerGroupHandler[T]) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	for msg := range claim.Messages() {
		event, err := h.source.decoder.Decode(msg.Value)
		if err != nil {
			if h.source.onHandleError != nil {
				h.source.onHandleError(err)
			}
			session.MarkMessage(msg, "")
			continue
		}

		eventCtx := &handler.Context{
			Context:    h.ctx,
			Event:      event,
			ReceivedAt: time.Now(),
		}
		if err := h.source.base.Handler()(eventCtx); err != nil && h.source.onHandleError != nil {
			h.source.onHandleError(err)
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

// Start launches background Kafka consumption and returns immediately.
func (s *KafkaSource[T]) Start(ctx context.Context) error {
	return s.base.StartLoop(ctx, s.consumeLoop)
}

// Stop cancels consumption and closes the underlying consumer group.
func (s *KafkaSource[T]) Stop(ctx context.Context) error {
	if err := s.base.Stop(ctx); err != nil {
		return err
	}
	return s.consumer.Close()
}

func (s *KafkaSource[T]) consumeLoop(ctx context.Context) {
	handler := &consumerGroupHandler[T]{
		source: s,
		ctx:    ctx,
	}

	for ctx.Err() == nil {
		if err := s.consumer.Consume(ctx, s.topics, handler); err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("[kafka-source] stopped")
				return
			}
			if s.onHandleError != nil {
				s.onHandleError(err)
				continue
			}
			log.Printf("[kafka-source] consume error: %v", err)
		}
	}
}
