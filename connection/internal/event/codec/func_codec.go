package codec

import "errors"

type FuncEventCodec[T any] struct {
	encodeFn func(event T) ([]byte, error)
	decodeFn func(data []byte) (T, error)
}

func NewFuncEventCodec[T any](
	encodeFn func(event T) ([]byte, error),
	decodeFn func(data []byte) (T, error),
) (EventCodec[T], error) {
	if encodeFn == nil || decodeFn == nil {
		return nil, errors.New("encodeFn and decodeFn are required")
	}
	return FuncEventCodec[T]{
		encodeFn: encodeFn,
		decodeFn: decodeFn,
	}, nil
}

func (c FuncEventCodec[T]) Encode(event T) ([]byte, error) {
	return c.encodeFn(event)
}

func (c FuncEventCodec[T]) Decode(data []byte) (T, error) {
	return c.decodeFn(data)
}
