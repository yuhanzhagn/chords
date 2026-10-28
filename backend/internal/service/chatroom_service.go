package service

import (
	"errors"

	"backend/internal/identity"
	"backend/internal/model"
	"backend/internal/repo"
	"gorm.io/gorm"
)

var ErrChatRoomNotFound = errors.New("chat room not found")

type chatRoomService struct {
	repos *repo.RepoContainer
}

func NewChatRoomService(repos *repo.RepoContainer) *chatRoomService {
	return &chatRoomService{repos: repos}
}

type ChatRoomService interface {
	CreateChatRoom(p identity.Principal, name string) (*model.ChatRoom, error)
	GetAllChatRooms() ([]model.ChatRoom, error)
	GetChatRoomByID(id uint) (*model.ChatRoom, error)
	DeleteChatRoom(p identity.Principal, id uint) error
	GetChatRoomByName(name string) (*model.ChatRoom, error)
	SearchChatRoomsByName(keyword string) ([]model.ChatRoom, error)
}

func (s *chatRoomService) CreateChatRoom(p identity.Principal, name string) (*model.ChatRoom, error) {
	if err := requireSession(s.repos, p); err != nil {
		return nil, err
	}
	existing, err := s.repos.ChatRoom.GetByName(name)
	if err == nil && existing != nil {
		return nil, errors.New("chat room already exists")
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	creatorID := p.UserID()
	chatRoom := &model.ChatRoom{Name: name, CreatorID: &creatorID}
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
			return nil, ErrChatRoomNotFound
		}
		return nil, err
	}
	return chatRoom, nil
}

func (s *chatRoomService) DeleteChatRoom(p identity.Principal, id uint) error {
	if err := requireSession(s.repos, p); err != nil {
		return err
	}
	room, err := s.repos.ChatRoom.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrChatRoomNotFound
		}
		return err
	}
	if room.CreatorID == nil || *room.CreatorID == 0 || *room.CreatorID != p.UserID() {
		return ErrForbidden
	}
	affected, err := s.repos.ChatRoom.DeleteOwned(id, p.UserID())
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrForbidden
	}
	return nil
}

func (s *chatRoomService) GetChatRoomByName(name string) (*model.ChatRoom, error) {
	if name == "" {
		return nil, errors.New("chat room name is required")
	}
	chatRoom, err := s.repos.ChatRoom.GetByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChatRoomNotFound
		}
		return nil, err
	}
	return chatRoom, nil
}

func (s *chatRoomService) SearchChatRoomsByName(keyword string) ([]model.ChatRoom, error) {
	return s.repos.ChatRoom.SearchByName(keyword)
}
