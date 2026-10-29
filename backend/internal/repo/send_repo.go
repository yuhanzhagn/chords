package repo

import (
	"backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SendRepo interface {
	FindRequest(userID uint, requestID string) (*model.SendRequest, error)
	InsertRequest(*model.SendRequest) (bool, error)
	InsertOutbox(*model.MessageOutbox) error
	FindOutbox(eventID string) (*model.MessageOutbox, error)
}
type sendRepo struct{ db *gorm.DB }

func NewSendRepo(db *gorm.DB) SendRepo { return &sendRepo{db: db} }
func (r *sendRepo) FindRequest(userID uint, requestID string) (*model.SendRequest, error) {
	var v model.SendRequest
	err := r.db.Where("user_id = ? AND request_id = ?", userID, requestID).First(&v).Error
	return &v, err
}
func (r *sendRepo) InsertRequest(v *model.SendRequest) (bool, error) {
	result := r.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "request_id"}}, DoNothing: true}).Create(v)
	return result.RowsAffected == 1, result.Error
}
func (r *sendRepo) InsertOutbox(v *model.MessageOutbox) error { return r.db.Create(v).Error }
func (r *sendRepo) FindOutbox(id string) (*model.MessageOutbox, error) {
	var v model.MessageOutbox
	err := r.db.First(&v, "event_id = ?", id).Error
	return &v, err
}
