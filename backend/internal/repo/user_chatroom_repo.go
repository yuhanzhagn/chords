package repo

import (
	"backend/internal/model"
)

// UserChatRoomRepo defines persistence for user-chatroom membership.
type UserChatRoomRepo interface {
	Create(m *model.UserChatRoom) error
	Exists(userID, chatRoomID uint) (bool, error)
	GetChatRoomsByUserID(userID uint) ([]model.ChatRoom, error)
}

type userChatRoomRepo struct {
	db gormDB
}

// NewUserChatRoomRepo returns a GORM-backed UserChatRoomRepo.
func NewUserChatRoomRepo(db gormDB) UserChatRoomRepo {
	return &userChatRoomRepo{db: db}
}

func (r *userChatRoomRepo) Create(m *model.UserChatRoom) error {
	return r.db.Create(m).Error
}

func (r *userChatRoomRepo) Exists(userID, chatRoomID uint) (bool, error) {
	var count int64
	err := r.db.Model(&model.UserChatRoom{}).
		Where("user_id = ? AND chat_room_id = ?", userID, chatRoomID).
		Count(&count).Error
	return count > 0, err
}

func (r *userChatRoomRepo) GetChatRoomsByUserID(userID uint) ([]model.ChatRoom, error) {
	var rooms []model.ChatRoom
	err := r.db.Table("chat_rooms").
		Select("chat_rooms.*").
		Joins("JOIN user_chat_rooms ON user_chat_rooms.chat_room_id = chat_rooms.id").
		Where("user_chat_rooms.user_id = ?", userID).
		Find(&rooms).Error
	return rooms, err
}
