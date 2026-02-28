package handler

import (
	"connection/internal/event/codec"
	"context"
	"errors"
	"fmt"
)

// FanoutHub is the minimum hub contract required by fanout egress handling.
// Broadcast enqueues payloads to client send channels, which are drained by writePump.
type FanoutHub[T any] interface {
	Codec() codec.EventCodec[T]
	GroupID(event T) uint32
	Broadcast(groupID uint32, msg []byte)
}

// FanoutHandler encodes outbound events and fans them out to group subscribers.
type FanoutHandler[T any] struct {
	hub FanoutHub[T]
}

func NewFanoutHandler[T any](hub FanoutHub[T]) (*FanoutHandler[T], error) {
	if hub == nil {
		return nil, errors.New("hub is required")
	}
	if hub.Codec() == nil {
		return nil, errors.New("hub codec is required")
	}
	return &FanoutHandler[T]{hub: hub}, nil
}

func (h *FanoutHandler[T]) Handle(ctx context.Context, event T) error {
	if h == nil || h.hub == nil {
		return errors.New("fanout handler is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	payload, err := h.hub.Codec().Encode(event)
	if err != nil {
		return fmt.Errorf("encode outbound event: %w", err)
	}

	h.hub.Broadcast(h.hub.GroupID(event), payload)
	return nil
}

// EgressHandler is the outbound entry point and contains the fanout handler.
type EgressHandler[T any] struct {
	fanout *FanoutHandler[T]
}

func NewEgressHandler[T any](hub FanoutHub[T]) (*EgressHandler[T], error) {
	fanout, err := NewFanoutHandler(hub)
	if err != nil {
		return nil, err
	}
	return &EgressHandler[T]{fanout: fanout}, nil
}

func (h *EgressHandler[T]) Fanout() *FanoutHandler[T] {
	if h == nil {
		return nil
	}
	return h.fanout
}

func (h *EgressHandler[T]) Handle(ctx context.Context, event T) error {
	if h == nil || h.fanout == nil {
		return errors.New("egress handler is not initialized")
	}
	return h.fanout.Handle(ctx, event)
}
