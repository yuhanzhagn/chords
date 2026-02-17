package codec

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type JSONEventCodec[T any] struct{}

func NewJSONEventCodec[T any]() EventCodec[T] {
	return JSONEventCodec[T]{}
}

func (JSONEventCodec[T]) Encode(event T) ([]byte, error) {
	return json.Marshal(event)
}

func (JSONEventCodec[T]) Decode(data []byte) (T, error) {
	var event T
	if err := json.Unmarshal(data, &event); err != nil {
		var zero T
		return zero, err
	}
	return event, nil
}

func (JSONEventCodec[T]) WSMessageType() int {
	return websocket.TextMessage
}
