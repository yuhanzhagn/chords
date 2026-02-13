package kafka

import (
	"encoding/json"
)

// kafka/event.go
type KafkaEvent struct {
	MsgID    uint64          `json:"ID"`
	UserID   uint            `json:"UserID"` // room:1 / user:123
	RoomID   uint            `json:"RoomID"`
	MsgType  string          `json:"MsgType"` // chat.message / notify
	Content  json.RawMessage `json:"Content"`
	TempID   string          `json:"TempID"`
	CreateAt int64           `json:"CreateAt"`
}
