package kafka

import (
	"context"
	"errors"

	"connection/internal/event/codec"
	"connection/internal/sink"

	"github.com/IBM/sarama"
)

// EventSink encodes events and publishes them to Kafka.
type EventSink[T any] struct {
	producer sarama.SyncProducer
	topic    string
	codec    codec.EventCodec[T]
}

func NewEventSink[T any](brokers []string, topic string, eventCodec codec.EventCodec[T]) (*EventSink[T], error) {
	if len(brokers) == 0 {
		return nil, errors.New("brokers are required")
	}
	if topic == "" {
		return nil, errors.New("topic is required")
	}
	if eventCodec == nil {
		return nil, errors.New("event codec is required")
	}

	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &EventSink[T]{
		producer: producer,
		topic:    topic,
		codec:    eventCodec,
	}, nil
}

func (s *EventSink[T]) Write(ctx context.Context, event T) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := s.codec.Encode(event)
	if err != nil {
		return err
	}

	_, _, err = s.producer.SendMessage(&sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(payload),
	})
	return err
}

func (s *EventSink[T]) Close() error {
	if s.producer == nil {
		return nil
	}
	return s.producer.Close()
}

var _ sink.Sink[any] = (*EventSink[any])(nil)
