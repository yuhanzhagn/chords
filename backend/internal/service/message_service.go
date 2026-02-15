package service

import (
	"backend/internal/model"
	"backend/internal/repo"
	"errors"
)

type messageService struct {
	repos *repo.RepoContainer
}

func NewMessageService(repos *repo.RepoContainer) *messageService {
	return &messageService{repos: repos}
}

type MessageService interface {
	CreateMessage(userID, roomID uint, content string) (*model.Message, error)
	GetMessagesByChatRoom(chatRoomID uint) ([]model.Message, error)
	DeleteMessage(id uint) error
	GetMessagesWithLimit(roomID uint, limit int) ([]model.Message, error)
}

func (s *messageService) CreateMessage(userID, roomID uint, content string) (*model.Message, error) {
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

func (s *messageService) GetMessagesByChatRoom(chatRoomID uint) ([]model.Message, error) {
	return s.repos.Message.GetByRoomID(chatRoomID)
}

func (s *messageService) DeleteMessage(id uint) error {
	rows, err := s.repos.Message.Delete(id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("message not found")
	}
	return nil
}

func (s *messageService) GetMessagesWithLimit(roomID uint, limit int) ([]model.Message, error) {
	return s.repos.Message.GetByRoomIDWithLimit(roomID, limit)
}
