package kafka

import (
	"encoding/json"
)

// kafka/event.go
type KafkaEvent struct {
	Topic   string          `json:"topic"` // room:1 / user:123
	Type    string          `json:"type"`  // chat.message / notify
	Payload json.RawMessage `json:"payload"`
}
