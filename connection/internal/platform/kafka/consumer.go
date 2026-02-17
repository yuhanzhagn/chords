// kafka/consumer.go
package kafka

import (
	"connection/internal/platform/codec"
	"context"
	"errors"
	"log"

	"github.com/IBM/sarama"
)

type WsOutboundConsumer[T any] struct {
	consumer sarama.ConsumerGroup
	groupID  string
	topics   []string
	handler  func(event T)
	decoder  codec.EventDecoder[T]
}

func NewWsOutboundConsumer[T any](
	brokers []string,
	groupID string,
	topics []string,
	handler func(event T),
	decoder codec.EventDecoder[T],
) (*WsOutboundConsumer[T], error) {
	if handler == nil {
		return nil, errors.New("handler is required")
	}
	if decoder == nil {
		return nil, errors.New("decoder is required")
	}

	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0

	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	cg, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &WsOutboundConsumer[T]{
		consumer: cg,
		groupID:  groupID,
		topics:   topics,
		handler:  handler,
		decoder:  decoder,
	}, nil
}

type wsOutboundHandler[T any] struct {
	handle  func(event T)
	decoder codec.EventDecoder[T]
}

func (h *wsOutboundHandler[T]) Setup(_ sarama.ConsumerGroupSession) error {
	log.Println("[kafka] ws outbound consumer setup")
	return nil
}

func (h *wsOutboundHandler[T]) Cleanup(_ sarama.ConsumerGroupSession) error {
	log.Println("[kafka] ws outbound consumer cleanup")
	return nil
}

func (h *wsOutboundHandler[T]) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {

	for msg := range claim.Messages() {

		event, err := h.decoder.Decode(msg.Value)
		if err != nil {
			log.Println("[kafka] decode error:", err)
			continue
		}

		h.handle(event)

		session.MarkMessage(msg, "")
	}

	return nil
}

func (c *WsOutboundConsumer[T]) Start(ctx context.Context) {
	handler := &wsOutboundHandler[T]{
		handle:  c.handler,
		decoder: c.decoder,
	}

	go func() {
		for {
			if err := c.consumer.Consume(ctx, c.topics, handler); err != nil {
				if errors.Is(err, context.Canceled) {
					log.Println("[kafka] consumer stopped")
					return
				}
				log.Println("[kafka] consume error:", err)
			}

		}
	}()
}

func (c *WsOutboundConsumer[T]) Close() error {
	return c.consumer.Close()
}
