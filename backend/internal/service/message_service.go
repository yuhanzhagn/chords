package service

import (
	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"errors"
	"gorm.io/gorm"
)

type messageService struct {
	repos *repo.RepoContainer
}

func NewMessageService(repos *repo.RepoContainer) *messageService {
	return &messageService{repos: repos}
}

type MessageService interface {
	CreateMessage(p identity.Principal, userID, roomID uint, content string) (*model.Message, error)
	GetMessagesByChatRoom(p identity.Principal, chatRoomID uint) ([]model.Message, error)
	DeleteMessage(p identity.Principal, id uint) error
	GetMessagesWithLimit(p identity.Principal, roomID uint, limit int) ([]model.Message, error)
	GetMessagesPage(p identity.Principal, roomID uint, beforeID uint, limit int) ([]model.Message, error)
}

func (s *messageService) CreateMessage(p identity.Principal, userID, roomID uint, content string) (*model.Message, error) {
	if userID != p.UserID() {
		return nil, ErrForbidden
	}
	if err := requireMember(s.repos, p, roomID); err != nil {
		return nil, err
	}
	msg := &model.Message{
		Content: content,
		UserID:  userID,
		RoomID:  roomID,
	}

	if err := s.repos.Message.Create(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *messageService) GetMessagesByChatRoom(p identity.Principal, chatRoomID uint) ([]model.Message, error) {
	if err := requireMember(s.repos, p, chatRoomID); err != nil {
		return nil, err
	}
	return s.repos.Message.GetByRoomID(chatRoomID)
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
	return s.repos.Message.GetByRoomIDWithLimit(roomID, limit)
}

func (s *messageService) GetMessagesPage(p identity.Principal, roomID uint, beforeID uint, limit int) ([]model.Message, error) {
	if err := requireMember(s.repos, p, roomID); err != nil {
		return nil, err
	}
	return s.repos.Message.GetByRoomIDBeforeWithLimit(roomID, beforeID, limit)
}
