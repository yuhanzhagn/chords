package model

import (
	"time"
)

type ChatRoom struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"unique;not null"`
	// CreatorID is nil for legacy rooms with unknown ownership. Never infer it.
	CreatorID *uint `gorm:"index"`
	CreatedAt time.Time
}
