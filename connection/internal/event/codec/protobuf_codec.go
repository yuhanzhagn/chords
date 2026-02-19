package codec

import (
	"errors"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type ProtobufEventCodec[T proto.Message] struct {
	newMessage func() T
}

func NewProtobufEventCodec[T proto.Message](newMessage func() T) (EventCodec[T], error) {
	if newMessage == nil {
		return nil, errors.New("newMessage is required")
	}
	return ProtobufEventCodec[T]{newMessage: newMessage}, nil
}

func (c ProtobufEventCodec[T]) Encode(event T) ([]byte, error) {
	return proto.Marshal(event)
}

func (c ProtobufEventCodec[T]) Decode(data []byte) (T, error) {
	event := c.newMessage()
	if err := proto.Unmarshal(data, event); err != nil {
		var zero T
		return zero, err
	}
	return event, nil
}

func (ProtobufEventCodec[T]) WSMessageType() int {
	return websocket.BinaryMessage
}
