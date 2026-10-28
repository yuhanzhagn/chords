package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"backend/internal/cache"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/utils"
	"gorm.io/gorm"
)

type authService struct {
	repos      *repo.RepoContainer
	cache      cache.Cache[BlockEntry]
	issueToken func(string, string) (string, string, time.Time, error)
}

type BlockEntry struct {
	JTI string
	Exp time.Time
}

type AuthService interface {
	Register(user *model.User) error
	Login(username, password string) (uint, string, string, string, error)
	Logout(tokenStr string) error
	ValidateJWT(tokenStr string) (*model.User, *model.UserSession, error)
	ForceLogoutAll(userID uint) error
	ForceLogoutOne(userID uint, sessionID string) error
	GetUserIDByUsername(username string) (uint, error)
	Block(jti string, exp time.Time) error
	IsBlocked(jti string) bool
	Clean(jti string)
}

func NewAuthService(repos *repo.RepoContainer, c cache.Cache[BlockEntry]) AuthService {
	return &authService{repos: repos, cache: c, issueToken: utils.GenerateJWT}
}

func (s *authService) Register(user *model.User) error {
	return createUser(s.repos, user)
}

func (s *authService) Login(username, password string) (uint, string, string, string, error) {
	user, err := s.repos.User.GetByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, "", "", "", errors.New("invalid username or password")
		}
		return 0, "", "", "", err
	}

	if !utils.CheckPasswordHash(password, user.Password) {
		return 0, "", "", "", errors.New("invalid username or password")
	}

	session := model.UserSession{
		UserID:     user.ID,
		SessionID:  generateSessionID(),
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}

	// Sign before changing persisted session state.
	token, jti, _, err := s.issueToken(user.Username, session.SessionID)
	if err != nil {
		return 0, "", "", "", err
	}
	if err := s.repos.UserSession.ReplaceActive(&session); err != nil {
		return 0, "", "", "", err
	}

	return user.ID, token, jti, session.SessionID, nil
}

func (s *authService) Logout(tokenStr string) error {
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil {
		return errors.New("invalid token")
	}

	jti := claims.ID
	if jti == "" {
		return errors.New("missing jti in token")
	}
	if claims.ExpiresAt == nil {
		return errors.New("missing exp in token")
	}
	exp := claims.ExpiresAt.Time

	if err := s.Block(jti, exp); err != nil {
		return errors.New("failed to block token")
	}
	userID, err := s.GetUserIDByUsername(claims.Username)
	if err != nil {
		return err
	}
	return s.ForceLogoutOne(userID, claims.SessionID)
}

func (s *authService) ValidateJWT(tokenStr string) (*model.User, *model.UserSession, error) {
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil {
		return nil, nil, errors.New("invalid token")
	}

	username := claims.Username
	userID, err := s.GetUserIDByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("username is invalid")
		}
		return nil, nil, fmt.Errorf("%w: %w", ErrAuthenticationUnavailable, err)
	}
	sessionID := claims.SessionID

	session, err := s.repos.UserSession.FindValidSession(sessionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("session invalid or revoked")
		}
		return nil, nil, fmt.Errorf("%w: %w", ErrAuthenticationUnavailable, err)
	}

	user, err := s.repos.User.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("user not found")
		}
		return nil, nil, fmt.Errorf("%w: %w", ErrAuthenticationUnavailable, err)
	}

	return user, session, nil
}

func (s *authService) ForceLogoutAll(userID uint) error {
	return s.repos.UserSession.RevokeAllByUserID(userID)
}

func (s *authService) ForceLogoutOne(userID uint, sessionID string) error {
	return s.repos.UserSession.RevokeOne(userID, sessionID)
}

func (s *authService) Block(jti string, exp time.Time) error {
	s.cache.Set(jti, BlockEntry{JTI: jti, Exp: exp}, time.Until(exp))
	return nil
}

func (s *authService) IsBlocked(jti string) bool {
	if _, ok := s.cache.Get(jti); ok {
		return true
	}
	return false
}

func (s *authService) Clean(jti string) {
	go func() {
		s.cache.Delete(jti)
	}()
}

func generateSessionID() string { return rand.Text() }

func (s *authService) GetUserIDByUsername(username string) (uint, error) {
	user, err := s.repos.User.GetByUsername(username)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}
