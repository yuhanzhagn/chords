package service

import (
	"errors"
	"log"
	"time"

	"backend/internal/cache"
	"backend/internal/model"
	"backend/internal/repo"
	"gorm.io/gorm"
)

type membershipService struct {
	repos *repo.RepoContainer
	cache cache.Cache[[]model.ChatRoom]
}

func NewMembershipService(repos *repo.RepoContainer, c cache.Cache[[]model.ChatRoom]) *membershipService {
	return &membershipService{repos: repos, cache: c}
}

type MembershipService interface {
	AddUserToChatRoom(username string, chatRoomID uint) error
	GetUserSubscribedChatRooms(username string) ([]model.ChatRoom, error)
	GetUserChatRoomsFromDB(username string) ([]model.ChatRoom, error)
}

func (s *membershipService) AddUserToChatRoom(username string, chatRoomID uint) error {
	user, err := s.repos.User.GetByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	exists, err := s.repos.ChatRoom.ExistsByID(chatRoomID)
	if err != nil || !exists {
		return errors.New("chatroom not found")
	}

	ok, err := s.repos.UserChatRoom.Exists(user.ID, chatRoomID)
	if err != nil {
		return err
	}
	if ok {
		return errors.New("user already in chatroom")
	}

	membership := model.UserChatRoom{
		UserID:     user.ID,
		ChatRoomID: chatRoomID,
		JoinedAt:   time.Now(),
	}
	if err := s.repos.UserChatRoom.Create(&membership); err != nil {
		return err
	}

	rooms, _ := s.GetUserChatRoomsFromDB(username)
	s.cache.Set(username, rooms, 30*time.Minute)
	return nil
}

func (s *membershipService) GetUserSubscribedChatRooms(username string) ([]model.ChatRoom, error) {
	rooms, isHit := s.cache.Get(username)
	if isHit {
		log.Println("Cache hit")
		return rooms, nil
	}

	rooms, err := s.GetUserChatRoomsFromDB(username)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := s.cache.Set(username, rooms, 30*time.Minute); err != nil {
			log.Println("cache set failed:", err)
		}
		log.Println("Add user to redis cache")
	}()
	return rooms, nil
}

func (s *membershipService) GetUserChatRoomsFromDB(username string) ([]model.ChatRoom, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}

	user, err := s.repos.User.GetByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return s.repos.UserChatRoom.GetChatRoomsByUserID(user.ID)
}
