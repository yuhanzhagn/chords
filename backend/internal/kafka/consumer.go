// kafka/consumer.go
package kafka

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	kafkapb "backend/proto/kafka"
)

type WsOutboundConsumer struct {
	consumer sarama.ConsumerGroup
	groupID  string
	topics   []string
	handler  func(event *kafkapb.KafkaEvent, commandToken string)
}

func NewWsOutboundConsumer(
	brokers []string,
	groupID string,
	topics []string,
	handler func(event *kafkapb.KafkaEvent, commandToken string),
) (*WsOutboundConsumer, error) {

	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	cg, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &WsOutboundConsumer{
		consumer: cg,
		groupID:  groupID,
		topics:   topics,
		handler:  handler,
	}, nil
}

type wsOutboundHandler struct {
	handle func(event *kafkapb.KafkaEvent, commandToken string)
}

func (h *wsOutboundHandler) Setup(_ sarama.ConsumerGroupSession) error {
	log.Println("[kafka] ws outbound consumer setup")
	return nil
}

func (h *wsOutboundHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	log.Println("[kafka] ws outbound consumer cleanup")
	return nil
}

func (h *wsOutboundHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {

	for msg := range claim.Messages() {
		var event kafkapb.KafkaEvent
		if err := proto.Unmarshal(msg.Value, &event); err != nil {
			log.Println("[kafka] unmarshal error:", err)
			continue
		}

		commandToken := ""
		for _, header := range msg.Headers {
			if string(header.Key) == "chords-command" {
				commandToken = string(header.Value)
			}
		}
		h.handle(&event, commandToken)

		session.MarkMessage(msg, "")
	}

	return nil
}

func (c *WsOutboundConsumer) Start(ctx context.Context) {
	handler := &wsOutboundHandler{
		handle: c.handler,
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

func (c *WsOutboundConsumer) Close() error {
	return c.consumer.Close()
}
