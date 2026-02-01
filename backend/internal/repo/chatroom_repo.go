package repo

import (
	"backend/internal/model"
)

// ChatRoomRepo defines persistence for chat rooms.
type ChatRoomRepo interface {
	Create(room *model.ChatRoom) error
	GetByID(id uint) (*model.ChatRoom, error)
	GetByName(name string) (*model.ChatRoom, error)
	GetAll() ([]model.ChatRoom, error)
	Delete(id uint) error
	SearchByName(keyword string) ([]model.ChatRoom, error)
	ExistsByID(id uint) (bool, error)
}

type chatRoomRepo struct {
	db gormDB
}

// NewChatRoomRepo returns a GORM-backed ChatRoomRepo.
func NewChatRoomRepo(db gormDB) ChatRoomRepo {
	return &chatRoomRepo{db: db}
}

func (r *chatRoomRepo) Create(room *model.ChatRoom) error {
	return r.db.Create(room).Error
}

func (r *chatRoomRepo) GetByID(id uint) (*model.ChatRoom, error) {
	var room model.ChatRoom
	if err := r.db.First(&room, id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *chatRoomRepo) GetByName(name string) (*model.ChatRoom, error) {
	var room model.ChatRoom
	if err := r.db.Where("name = ?", name).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *chatRoomRepo) GetAll() ([]model.ChatRoom, error) {
	var rooms []model.ChatRoom
	if err := r.db.Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *chatRoomRepo) Delete(id uint) error {
	return r.db.Delete(&model.ChatRoom{}, id).Error
}

func (r *chatRoomRepo) SearchByName(keyword string) ([]model.ChatRoom, error) {
	var rooms []model.ChatRoom
	pattern := "%" + keyword + "%"
	if err := r.db.Where("name LIKE ?", pattern).Find(&rooms).Error; err != nil {
		return nil, err
	}
	return rooms, nil
}

func (r *chatRoomRepo) ExistsByID(id uint) (bool, error) {
	var exists bool
	err := r.db.Model(&model.ChatRoom{}).
		Select("count(*) > 0").
		Where("id = ?", id).
		Scan(&exists).Error
	return exists, err
}
