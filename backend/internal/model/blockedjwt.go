package model

import "time"

type BlockedJWT struct {
	ID        uint   `gorm:"primaryKey"`
	JTI       string `gorm:"uniqueIndex"`
	ExpiresAt time.Time
	CreatedAt time.Time
}
