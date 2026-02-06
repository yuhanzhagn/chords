package service

import (
	"backend/internal/kafka"
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

type KafkaService struct {
	Producer       MessageProducer
	MessageService MessageService
}

func (s *KafkaService) HandleOutboundEvent(event kafka.KafkaEvent) {
	// Here you can add logic like logging or basic validation
	log.Printf("Handling outbound event: Topic=%s, Type=%s", event.Topic, event.Type)
	switch event.Type {
	case "chat.message":
		s.handleChatMessage(event)
	case "room.join":
		s.handleJoin(event)
	case "room.leave":
		s.handleLeave(event)
	default:
		// Unknown event type
	}
}

func (s *KafkaService) handleChatMessage(event kafka.KafkaEvent) {
	// Process chat message event
	// For example, you might want to log it or transform it before publishing
	//
	s.HandleOutgoingMessage(event.Payload)
}

func (s *KafkaService) handleJoin(event kafka.KafkaEvent) {
	// Process room join event
	// For example, you might want to log it or transform it before publishing
	s.HandleOutgoingMessage(event.Payload)
}

func (s *KafkaService) handleLeave(event kafka.KafkaEvent) {
	// Process room leave event
	// For example, you might want to log it or transform it before publishing
	s.HandleOutgoingMessage(event.Payload)
}

// HandleOutgoingMessage handles messages consumed from Kafka
func (s *KafkaService) HandleOutgoingMessage(data []byte) error {
	return s.Producer.Publish("ws.inbound", nil, data)
}
