package service

import (
	"backend/internal/identity"
	"backend/utils"
	"log"

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
	if len(commandToken) > kafkapb.MaxCommandHeaderBytes || kafkapb.ValidateSend(event) != nil {
		log.Println("rejected oversized or invalid command")
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
	result, err := s.MessageService.SendMessage(p, SendInput{UserID: uint(event.UserId), RoomID: uint(event.RoomId), Content: string(event.Content), RequestID: event.RequestId, TempID: event.TempId, ClientID: event.Id})
	if err != nil {
		log.Println("message command rejected:", err)
		return
	}
	// Optional adapter producer for injected callers; production shares the service producer.
	if !result.Deleted && s.Producer != nil {
		if err := s.MessageService.PublishPending(result.EventID, s.Producer); err != nil {
			log.Println("message persisted; outbox publish remains pending")
		}
	}
}
