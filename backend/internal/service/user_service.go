package service

import (
	"errors"

	"backend/internal/model"
	"backend/internal/repo"
	"gorm.io/gorm"
)

type userService struct {
	repos *repo.RepoContainer
}

func NewUserService(repos *repo.RepoContainer) *userService {
	return &userService{repos: repos}
}

type UserService interface {
	CreateUser(user *model.User) error
	GetAllUsers() ([]model.User, error)
	GetUserByUsername(username string) (*model.User, error)
}

func (s *userService) CreateUser(user *model.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.Username == "" {
		return errors.New("user name is required")
	}
	return s.repos.User.Create(user)
}

func (s *userService) GetAllUsers() ([]model.User, error) {
	return s.repos.User.GetAll()
}

func (s *userService) GetUserByUsername(username string) (*model.User, error) {
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
	return user, nil
}
