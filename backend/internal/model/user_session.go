package model

import (
    "time"

    "gorm.io/gorm"
)

type UserSession struct {
    ID             uint           `gorm:"primaryKey"`
    UserID         uint           `gorm:"index;not null"`       // FK to User
    SessionID      string         `gorm:"uniqueIndex;not null"` // JWT session ID
    SessionVersion int            `gorm:"default:1"`
    DeviceInfo     string
    RefreshToken   string
    Revoked        bool           `gorm:"default:false"`
    CreatedAt      time.Time
    LastUsedAt     time.Time
    ExpiresAt      time.Time
    DeletedAt      gorm.DeletedAt `gorm:"index"`

    // Optional: GORM foreign key constraint
    User           User           `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

