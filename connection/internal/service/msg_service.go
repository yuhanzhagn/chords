package service

import (
	"log"
)

// MessageProducer defines the contract for sending messages to a broker
type MessageProducer interface {
	Publish(topic string, key []byte, value []byte) error
	Close() error
}

type OutgoingHandler interface {
	Deliver(data []byte) error
}

type MessageService struct {
	Producer        MessageProducer
	OutgoingHandler OutgoingHandler
}

func (s *MessageService) HandleIncomingMessage(data []byte) {
	// Here you can add logic like logging or basic validation
	log.Printf("Handling incoming message: %s", string(data))
	err := s.Producer.Publish("ws.inbound", nil, data)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	}
}

// HandleOutgoingMessage handles messages consumed from Kafka
func (s *MessageService) HandleOutgoingMessage(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return s.OutgoingHandler.Deliver(data)
}
