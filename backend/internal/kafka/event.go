package kafka

import (
	"encoding/json"
)

// kafka/event.go
type KafkaEvent struct {
	UserID  uint            `json:"UserID"` // room:1 / user:123
	RoomID  uint            `json:"RoomID"`
	MsgType string          `json:"MsgType"` // chat.message / notify
	Content json.RawMessage `json:"Content"`
	TempID  string          `json:"TempID"`
}
