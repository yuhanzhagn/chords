package model

import (
	"time"
)

type UserChatRoom struct {
	ID         uint `gorm:"primaryKey"`
	UserID     uint `gorm:"not null;index"`
	ChatRoomID uint `gorm:"not null;index"`
	JoinedAt   time.Time
}
