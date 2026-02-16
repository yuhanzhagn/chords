package service

import (
	"log"
	"time"

	kafkapb "backend/proto/kafka"

	"google.golang.org/protobuf/proto"
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

func (s *KafkaService) HandleOutboundEvent(event *kafkapb.KafkaEvent) {
	// Here you can add logic like logging or basic validation
	log.Printf("Handling outbound event: UserID=%d, RoomID=%d, MsgType=%s", event.UserId, event.RoomId, event.MsgType)
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

func (s *KafkaService) handleChatMessage(event *kafkapb.KafkaEvent) {
	// Process chat message event
	// For example, you might want to log it or transform it before publishing
	//
	_, err := s.MessageService.CreateMessage(uint(event.UserId), uint(event.RoomId), string(event.Content))
	if err != nil {
		log.Println("Error creating message:", err)
	}

	s.HandleOutgoingMessage(event)
}

func (s *KafkaService) handleJoin(event *kafkapb.KafkaEvent) {
	// Process room join event
	// For example, you might want to log it or transform it before publishing
	s.HandleOutgoingMessage(event)
}

func (s *KafkaService) handleLeave(event *kafkapb.KafkaEvent) {
	// Process room leave event
	// For example, you might want to log it or transform it before publishing

	s.HandleOutgoingMessage(event)
}

// HandleOutgoingMessage handles messages consumed from Kafka
func (s *KafkaService) HandleOutgoingMessage(event *kafkapb.KafkaEvent) error {
	event.CreatedAt = time.Now().Unix()
	rawbyte, err := proto.Marshal(event)
	if err != nil {
		log.Println("Error marshaling event:", err)
		return err
	}
	return s.Producer.Publish("notification", nil, rawbyte)
}
