package service

import (
	"backend/internal/identity"
	"backend/internal/repo"
	"errors"
	"gorm.io/gorm"
)

var ErrAuthenticationUnavailable = errors.New("authentication service unavailable")

var ErrForbidden = errors.New("forbidden")
var ErrUnauthenticated = errors.New("authentication required")

func requireSession(repos *repo.RepoContainer, p identity.Principal) error {
	if !p.Valid() {
		return ErrUnauthenticated
	}
	if _, err := repos.UserSession.FindValidSession(p.SessionID(), p.UserID()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUnauthenticated
		}
		return err
	}
	return nil
}

func requireMember(repos *repo.RepoContainer, p identity.Principal, roomID uint) error {
	if err := requireSession(repos, p); err != nil {
		return err
	}
	exists, err := repos.ChatRoom.ExistsByID(roomID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrForbidden
	}
	member, err := repos.UserChatRoom.Exists(p.UserID(), roomID)
	if err != nil {
		return err
	}
	if !member {
		return ErrForbidden
	}
	return nil
}
