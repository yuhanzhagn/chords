package service

import (
	"backend/internal/kafka"
	"encoding/json"
	"log"
	"time"
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
	log.Printf("Handling outbound event: UserID=%s, RoomID=%s, MsgType=%s", event.UserID, event.RoomID, event.MsgType)
	switch event.MsgType {
	case "message":
		s.handleChatMessage(event)
	case "join":
		s.handleJoin(event)
	case "leave":
		s.handleLeave(event)
	default:
		// Unknown event type
	}
}

func (s *KafkaService) handleChatMessage(event kafka.KafkaEvent) {
	// Process chat message event
	// For example, you might want to log it or transform it before publishing
	//
	s.HandleOutgoingMessage(event)
}

func (s *KafkaService) handleJoin(event kafka.KafkaEvent) {
	// Process room join event
	// For example, you might want to log it or transform it before publishing
	s.HandleOutgoingMessage(event)
}

func (s *KafkaService) handleLeave(event kafka.KafkaEvent) {
	// Process room leave event
	// For example, you might want to log it or transform it before publishing

	s.HandleOutgoingMessage(event)
}

// HandleOutgoingMessage handles messages consumed from Kafka
func (s *KafkaService) HandleOutgoingMessage(event kafka.KafkaEvent) error {
	event.CreateAt = time.Now().Unix()
	rawbyte, err := json.Marshal(event)
	if err != nil {
		log.Println("Error marshaling event:", err)
		return err
	}
	return s.Producer.Publish("ws.inbound", nil, rawbyte)
}
