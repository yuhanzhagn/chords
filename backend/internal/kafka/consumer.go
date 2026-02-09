// kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/IBM/sarama"
)

type WsOutboundConsumer struct {
	consumer sarama.ConsumerGroup
	groupID  string
	topics   []string
	handler  func(event KafkaEvent)
}

func NewWsOutboundConsumer(
	brokers []string,
	groupID string,
	topics []string,
	handler func(event KafkaEvent),
) (*WsOutboundConsumer, error) {

	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0

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
	handle func(event KafkaEvent)
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

		var event KafkaEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Println("[kafka] unmarshal error:", err)
			continue
		}
		log.Printf("[kafka] ws outbound consumed message: topic=%s, partition=%d, offset=%d, value=%s\n",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Value))
		h.handle(event)

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
