package model

import (
	"time"
)

// User represents a chat user
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username" binding:"required"`
	Email     string    `gorm:"unique;not null" json:"email" binding:"required,email"`
	Password  string    `json:"password,omitempty" binding:"required"`
	CreatedAt time.Time `json:"created_at"`

	Sessions []UserSession `gorm:"foreignKey:UserID" json:"sessions,omitempty"`
}
