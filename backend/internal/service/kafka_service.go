package service

import (
	"backend/internal/identity"
	"backend/utils"
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

func (s *KafkaService) HandleOutboundEvent(event *kafkapb.KafkaEvent, commandToken string) {
	if event == nil || event.MsgType != "message" {
		return
	}
	claims, err := utils.ParseCommand(commandToken)
	if err != nil {
		log.Println("rejected unauthorized command")
		return
	}
	raw, err := proto.Marshal(event)
	if err != nil || claims.UserID != uint(event.UserId) || claims.RoomID != uint(event.RoomId) || claims.Digest != utils.EventDigest(raw) {
		log.Println("rejected altered command")
		return
	}
	p := identity.New(claims.UserID, claims.Username, claims.SessionID)
	s.handleChatMessage(p, event)
}

func (s *KafkaService) handleChatMessage(p identity.Principal, event *kafkapb.KafkaEvent) {
	_, err := s.MessageService.CreateMessage(p, uint(event.UserId), uint(event.RoomId), string(event.Content))
	if err != nil {
		log.Println("Error creating message:", err)
		return
	}

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
