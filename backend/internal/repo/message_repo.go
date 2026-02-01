package repo

import (
	"backend/internal/model"
	"gorm.io/gorm"
)

// MessageRepo defines persistence for messages.
type MessageRepo interface {
	Create(msg *model.Message) error
	GetByChatRoomID(chatRoomID uint) ([]model.Message, error)
	GetByChatRoomIDWithLimit(chatRoomID uint, limit int) ([]model.Message, error)
	Delete(id uint) (rowsAffected int64, err error)
}

type messageRepo struct {
	db *gorm.DB
}

// NewMessageRepo returns a GORM-backed MessageRepo.
func NewMessageRepo(db *gorm.DB) MessageRepo {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(msg *model.Message) error {
	return r.db.Create(msg).Error
}

func (r *messageRepo) GetByChatRoomID(chatRoomID uint) ([]model.Message, error) {
	var messages []model.Message
	if err := r.db.Where("chat_room_id = ?", chatRoomID).Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *messageRepo) GetByChatRoomIDWithLimit(chatRoomID uint, limit int) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.Where("chat_room_id = ?", chatRoomID).Order("created_at desc").Limit(limit).Find(&messages).Error
	return messages, err
}

func (r *messageRepo) Delete(id uint) (int64, error) {
	res := r.db.Delete(&model.Message{}, id)
	return res.RowsAffected, res.Error
}
