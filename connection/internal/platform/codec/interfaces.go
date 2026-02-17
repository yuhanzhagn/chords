package codec

type EventDecoder[T any] interface {
	Decode(data []byte) (T, error)
}

type EventEncoder[T any] interface {
	Encode(event T) ([]byte, error)
}

type EventCodec[T any] interface {
	EventDecoder[T]
	EventEncoder[T]
}

type WSMessageTypeProvider interface {
	WSMessageType() int
}
