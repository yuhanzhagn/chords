package kafka

import (
	"encoding/json"
)

// kafka/event.go
type KafkaEvent struct {
	UserID  uint            `json:"UserID"` // room:1 / user:123
	RoomID  uint            `json:"RoomID"`
	MsgType string          `json:"MsgType"` // chat.message / notify
	Message json.RawMessage `json:"Message"`
	TempID  string          `json:"TempID"`
}
