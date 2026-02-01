package repo

import (
	"backend/internal/model"
)

// UserSessionRepo defines persistence for user sessions.
type UserSessionRepo interface {
	Create(session *model.UserSession) error
	FindValidSession(sessionID string, userID uint) (*model.UserSession, error)
	RevokeAllByUserID(userID uint) error
	RevokeOne(userID uint, sessionID string) error
}

type userSessionRepo struct {
	db gormDB
}

// NewUserSessionRepo returns a GORM-backed UserSessionRepo.
func NewUserSessionRepo(db gormDB) UserSessionRepo {
	return &userSessionRepo{db: db}
}

func (r *userSessionRepo) Create(session *model.UserSession) error {
	return r.db.Create(session).Error
}

func (r *userSessionRepo) FindValidSession(sessionID string, userID uint) (*model.UserSession, error) {
	var session model.UserSession
	if err := r.db.Where("session_id = ? AND user_id = ? AND revoked = ?", sessionID, userID, false).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *userSessionRepo) RevokeAllByUserID(userID uint) error {
	return r.db.Model(&model.UserSession{}).Where("user_id = ?", userID).Update("revoked", true).Error
}

func (r *userSessionRepo) RevokeOne(userID uint, sessionID string) error {
	return r.db.Model(&model.UserSession{}).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Update("revoked", true).Error
}
