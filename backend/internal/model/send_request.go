package model

import "time"

// SendRequest is a permanent idempotency tombstone, independent of message deletion/session.
// Deliberately no message FK: deleting a message must never free a request key.
type SendRequest struct {
	UserID             uint   `gorm:"primaryKey;not null;autoIncrement:false"`
	RequestID          string `gorm:"primaryKey;not null;size:128"`
	Fingerprint        string `gorm:"not null"`
	FingerprintVersion uint   `gorm:"not null"`
	RoomID             uint   `gorm:"not null"`
	RoomSeq            uint64 `gorm:"not null;default:0"`
	MessageID          uint   `gorm:"not null"`
	EventID            string `gorm:"not null;uniqueIndex"`
	CreatedAt          time.Time
}

type MessageOutbox struct {
	EventID       string `gorm:"primaryKey;not null"`
	RoomID        uint   `gorm:"not null"`
	SchemaVersion uint   `gorm:"not null"`
	Payload       []byte `gorm:"not null"`
	Status        string `gorm:"not null;index:idx_outbox_pending"`
	CreatedAt     time.Time
	PublishedAt   *time.Time
	LeaseToken    string `gorm:"not null;default:''"`
	LeaseUntil    int64  `gorm:"not null;default:0"`
	Attempts      uint   `gorm:"not null;default:0"`
	NextAttempt   int64  `gorm:"not null;default:0"`
	LastError     string `gorm:"not null;default:''"`
}

func (MessageOutbox) TableName() string { return "message_outbox" }

type SchemaMigration struct {
	Version   uint      `gorm:"primaryKey;not null;autoIncrement:false"`
	AppliedAt time.Time `gorm:"not null"`
}

// HighWater survives deletion, including deletion of the last message.
type RoomSequence struct {
	RoomID    uint   `gorm:"primaryKey;autoIncrement:false"`
	HighWater uint64 `gorm:"not null"`
}
