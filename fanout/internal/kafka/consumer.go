package kafka

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	kafkapb "fanout/proto/kafka"
)

type NotificationConsumer struct {
	consumer sarama.ConsumerGroup
	groupID  string
	topics   []string
	handler  func(ctx context.Context, event *kafkapb.KafkaEvent) error
}

func NewNotificationConsumer(
	brokers []string,
	groupID string,
	topics []string,
	handler func(ctx context.Context, event *kafkapb.KafkaEvent) error,
) (*NotificationConsumer, error) {
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

	return &NotificationConsumer{
		consumer: cg,
		groupID:  groupID,
		topics:   topics,
		handler:  handler,
	}, nil
}

type notificationHandler struct {
	handle func(ctx context.Context, event *kafkapb.KafkaEvent) error
}

func (h *notificationHandler) Setup(_ sarama.ConsumerGroupSession) error {
	log.Println("[fanout-kafka] consumer setup")
	return nil
}

func (h *notificationHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	log.Println("[fanout-kafka] consumer cleanup")
	return nil
}

func (h *notificationHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ctx := session.Context()
	for msg := range claim.Messages() {
		var event kafkapb.KafkaEvent
		if err := proto.Unmarshal(msg.Value, &event); err != nil {
			log.Println("[fanout-kafka] unmarshal error:", err)
			session.MarkMessage(msg, "")
			continue
		}

		if err := h.handle(ctx, &event); err != nil {
			log.Printf("[fanout-kafka] handler error: %v", err)
		}

		session.MarkMessage(msg, "")
	}

	return nil
}

func (c *NotificationConsumer) Start(ctx context.Context) {
	handler := &notificationHandler{handle: c.handler}

	go func() {
		for {
			if err := c.consumer.Consume(ctx, c.topics, handler); err != nil {
				if errors.Is(err, context.Canceled) {
					log.Println("[fanout-kafka] consumer stopped")
					return
				}
				log.Println("[fanout-kafka] consume error:", err)
			}
		}
	}()
}

func (c *NotificationConsumer) Close() error {
	return c.consumer.Close()
}
