package service

import (
	"errors"
	"fmt"

	"backend/internal/model"
	"backend/internal/repo"
	"backend/utils"
	"gorm.io/gorm"
)

var ErrInvalidPassword = errors.New("invalid password")

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
	return createUser(s.repos, user)
}

// createUser accepts plaintext credentials; persistence is the only layer accepting hashes.
func createUser(repos *repo.RepoContainer, user *model.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.Username == "" {
		return errors.New("user name is required")
	}
	if user.Password == "" {
		return errors.New("password is required")
	}
	hash, err := utils.HashPassword(user.Password)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidPassword, err)
	}
	// Copy only writable fields, including for service callers.
	created := model.User{Username: user.Username, Email: user.Email, Password: hash}
	if err := repos.User.Create(&created); err != nil {
		return err
	}
	*user = created
	return nil
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
