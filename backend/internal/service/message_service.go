package service

import (
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"errors"
	"gorm.io/gorm"
	"strconv"
)

type messageService struct {
	repos    *repo.RepoContainer
	producer MessageProducer
}

func NewMessageService(repos *repo.RepoContainer, producers ...MessageProducer) *messageService {
	s := &messageService{repos: repos}
	if len(producers) > 0 {
		s.producer = producers[0]
	}
	return s
}

type MessageService interface {
	SendMessage(p identity.Principal, in SendInput) (*SendResult, error)
	PublishPending(eventID string, producer MessageProducer) error
	CreateMessage(p identity.Principal, userID, roomID uint, content string) (*model.Message, error)
	GetMessagesByChatRoom(p identity.Principal, chatRoomID uint) ([]model.Message, error)
	DeleteMessage(p identity.Principal, id uint) error
	GetMessagesWithLimit(p identity.Principal, roomID uint, limit int) ([]model.Message, error)
	GetMessagesPage(p identity.Principal, roomID uint, beforeID uint, limit int) ([]model.Message, error)
}

func (s *messageService) CreateMessage(p identity.Principal, userID, roomID uint, content string) (*model.Message, error) {
	result, err := s.SendMessage(p, SendInput{UserID: userID, RoomID: roomID, Content: content})
	if err != nil {
		return nil, err
	}
	return &model.Message{ID: result.ID, UserID: result.UserID, RoomID: result.RoomID, Content: result.Content, CreatedAt: result.CreatedAt, RoomSeq: mustRoomSeq(result.RoomSeq)}, nil
}

func (s *messageService) GetMessagesByChatRoom(p identity.Principal, chatRoomID uint) ([]model.Message, error) {
	if err := requireMember(s.repos, p, chatRoomID); err != nil {
		return nil, err
	}
	return s.repos.Message.GetByRoomIDBeforeWithLimit(chatRoomID, 0, 100)
}

func (s *messageService) DeleteMessage(p identity.Principal, id uint) error {
	if err := requireSession(s.repos, p); err != nil {
		return err
	}
	msg, err := s.repos.Message.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("message not found")
		}
		return err
	}
	if msg.UserID != p.UserID() {
		return ErrForbidden
	}
	if err := requireMember(s.repos, p, msg.RoomID); err != nil {
		return err
	}
	rows, err := s.repos.Message.DeleteOwned(id, p.UserID())
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("message not found")
	}
	return nil
}

func (s *messageService) GetMessagesWithLimit(p identity.Principal, roomID uint, limit int) ([]model.Message, error) {
	if err := requireMember(s.repos, p, roomID); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidCursor
	}
	return s.repos.Message.GetByRoomIDWithLimit(roomID, limit)
}

func (s *messageService) GetMessagesPage(p identity.Principal, roomID uint, beforeID uint, limit int) ([]model.Message, error) {
	if err := requireMember(s.repos, p, roomID); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, ErrInvalidCursor
	}
	return s.repos.Message.GetByRoomIDBeforeWithLimit(roomID, beforeID, limit)
}

func mustRoomSeq(s string) uint64 { n, _ := strconv.ParseUint(s, 10, 63); return n }
