package model

import (
	"encoding/json"
	"strconv"
	"time"
)

// Message represents a message sent in a chat room
type Message struct {
	ID        uint   `gorm:"primaryKey"`
	Content   string `gorm:"type:text;not null"`
	UserID    uint   `gorm:"not null"` // Foreign key to User
	RoomID    uint   `gorm:"not null"` // Foreign key to ChatRoom
	RoomSeq   uint64 `gorm:"not null;default:0" json:"-"`
	RequestID string `gorm:"->;-:migration" json:"request_id,omitempty"`
	EventID   string `gorm:"->;-:migration" json:"event_id,omitempty"`
	CreatedAt time.Time
}

// Add a precise decimal ID for new HTTP readers while retaining the legacy numeric ID.
func (m Message) MarshalJSON() ([]byte, error) {
	type legacy Message
	return json.Marshal(struct {
		legacy
		ServerMessageID string `json:"server_message_id"`
		RoomSeq         string `json:"room_seq"`
	}{legacy(m), strconv.FormatUint(uint64(m.ID), 10), strconv.FormatUint(m.RoomSeq, 10)})
}
