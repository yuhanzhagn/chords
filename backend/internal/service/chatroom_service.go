package service

import (
	"errors"

	"backend/internal/model"
	"backend/internal/repo"
	"gorm.io/gorm"
)

type chatRoomService struct {
	repos *repo.RepoContainer
}

func NewChatRoomService(repos *repo.RepoContainer) *chatRoomService {
	return &chatRoomService{repos: repos}
}

type ChatRoomService interface {
	CreateChatRoom(name string) (*model.ChatRoom, error)
	GetAllChatRooms() ([]model.ChatRoom, error)
	GetChatRoomByID(id uint) (*model.ChatRoom, error)
	DeleteChatRoom(id uint) error
	GetChatRoomByName(name string) (*model.ChatRoom, error)
	SearchChatRoomsByName(keyword string) ([]model.ChatRoom, error)
}

func (s *chatRoomService) CreateChatRoom(name string) (*model.ChatRoom, error) {
	existing, err := s.repos.ChatRoom.GetByName(name)
	if err == nil && existing != nil {
		return nil, errors.New("chat room already exists")
	}

	chatRoom := &model.ChatRoom{Name: name}
	if err := s.repos.ChatRoom.Create(chatRoom); err != nil {
		return nil, err
	}
	return chatRoom, nil
}

func (s *chatRoomService) GetAllChatRooms() ([]model.ChatRoom, error) {
	return s.repos.ChatRoom.GetAll()
}

func (s *chatRoomService) GetChatRoomByID(id uint) (*model.ChatRoom, error) {
	chatRoom, err := s.repos.ChatRoom.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("chat room not found")
		}
		return nil, err
	}
	return chatRoom, nil
}

func (s *chatRoomService) DeleteChatRoom(id uint) error {
	return s.repos.ChatRoom.Delete(id)
}

func (s *chatRoomService) GetChatRoomByName(name string) (*model.ChatRoom, error) {
	if name == "" {
		return nil, errors.New("chat room name is required")
	}
	chatRoom, err := s.repos.ChatRoom.GetByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("chat room not found")
		}
		return nil, err
	}
	return chatRoom, nil
}

func (s *chatRoomService) SearchChatRoomsByName(keyword string) ([]model.ChatRoom, error) {
	return s.repos.ChatRoom.SearchByName(keyword)
}
