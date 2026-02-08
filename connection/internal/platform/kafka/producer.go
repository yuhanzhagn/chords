package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    "ws.inbound",
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *KafkaProducer) Publish(topic string, key []byte, value []byte) error {
	return p.writer.WriteMessages(context.Background(), kafka.Message{
		Key:   key,
		Value: value,
	})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
