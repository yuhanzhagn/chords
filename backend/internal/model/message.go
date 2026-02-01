package model

import (
    "time"
)

// Message represents a message sent in a chat room
type Message struct {
    ID         uint      `gorm:"primaryKey"`
    Content    string    `gorm:"type:text;not null"`
    UserID     uint      `gorm:"not null"`  // Foreign key to User
    ChatRoomID uint      `gorm:"not null"`  // Foreign key to ChatRoom
    CreatedAt  time.Time
}
