package repo

import (
	"backend/internal/model"
	"gorm.io/gorm"
	"time"
)

// UserSessionRepo defines persistence for user sessions.
type UserSessionRepo interface {
	Create(session *model.UserSession) error
	ReplaceActive(session *model.UserSession) error
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
	if err := r.db.Where("session_id = ? AND user_id = ? AND revoked = ? AND expires_at > ?", sessionID, userID, false, time.Now()).First(&session).Error; err != nil {
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

// ReplaceActive serializes replacements through a write before inserting the new session.
// Both writes commit together, so failed inserts preserve all prior sessions.
func (r *userSessionRepo) ReplaceActive(session *model.UserSession) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserSession{}).Where("user_id = ? AND revoked = ?", session.UserID, false).Update("revoked", true).Error; err != nil {
			return err
		}
		return tx.Create(session).Error
	})
}
