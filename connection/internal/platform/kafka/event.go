package kafka

import (
	"encoding/json"
)

// kafka/event.go
type KafkaEvent1 struct {
	MsgID     uint64          `json:"ID"`
	UserID    uint32          `json:"UserID"` // room:1 / user:123
	RoomID    uint32          `json:"RoomID"`
	MsgType   string          `json:"MsgType"` // chat.message / notify
	Content   json.RawMessage `json:"Content"`
	TempID    string          `json:"TempID"`
	CreatedAt int64           `json:"CreateAt"`
}
