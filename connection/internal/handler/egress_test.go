package handler

import (
	"context"
	"errors"
	"testing"

	"connection/internal/event/codec"
)

type stubEvent struct {
	Room uint32
	Body string
}

type stubCodec struct {
	encoded []byte
	err     error
}

func (c *stubCodec) Decode(_ []byte) (stubEvent, error) {
	return stubEvent{}, nil
}

func (c *stubCodec) Encode(_ stubEvent) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.encoded, nil
}

type stubHub struct {
	codec codec.EventCodec[stubEvent]

	lastRoom uint32
	lastMsg  []byte
	calls    int
}

func (h *stubHub) Codec() codec.EventCodec[stubEvent] {
	return h.codec
}

func (h *stubHub) RoomID(event stubEvent) uint32 {
	return event.Room
}

func (h *stubHub) Broadcast(roomID uint32, msg []byte) {
	h.calls++
	h.lastRoom = roomID
	h.lastMsg = append([]byte(nil), msg...)
}

func TestNewFanoutHandler_ValidateInput(t *testing.T) {
	if _, err := NewFanoutHandler[stubEvent](nil); err == nil {
		t.Fatalf("expected nil hub validation error")
	}

	hub := &stubHub{}
	if _, err := NewFanoutHandler[stubEvent](hub); err == nil {
		t.Fatalf("expected nil codec validation error")
	}
}

func TestFanoutHandler_Handle_BroadcastToWritePumpChannel(t *testing.T) {
	hub := &stubHub{
		codec: &stubCodec{encoded: []byte("payload")},
	}
	fanout, err := NewFanoutHandler[stubEvent](hub)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	event := stubEvent{Room: 42, Body: "hello"}
	if err := fanout.Handle(context.Background(), event); err != nil {
		t.Fatalf("unexpected handle error: %v", err)
	}

	if hub.calls != 1 {
		t.Fatalf("expected single broadcast call, got %d", hub.calls)
	}
	if hub.lastRoom != 42 {
		t.Fatalf("expected room 42, got %d", hub.lastRoom)
	}
	if string(hub.lastMsg) != "payload" {
		t.Fatalf("expected encoded payload to be broadcast, got %q", string(hub.lastMsg))
	}
}

func TestFanoutHandler_Handle_EncodeError(t *testing.T) {
	expectedErr := errors.New("encode failed")
	hub := &stubHub{
		codec: &stubCodec{err: expectedErr},
	}
	fanout, err := NewFanoutHandler[stubEvent](hub)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	err = fanout.Handle(context.Background(), stubEvent{Room: 1})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected wrapped encode error, got %v", err)
	}
	if hub.calls != 0 {
		t.Fatalf("expected no broadcast on encode error, got %d", hub.calls)
	}
}

func TestEgressHandler_Handle_DelegatesFanout(t *testing.T) {
	hub := &stubHub{
		codec: &stubCodec{encoded: []byte("ok")},
	}
	egress, err := NewEgressHandler[stubEvent](hub)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if egress.Fanout() == nil {
		t.Fatalf("expected fanout handler to be initialized")
	}

	if err := egress.Handle(context.Background(), stubEvent{Room: 7}); err != nil {
		t.Fatalf("unexpected egress handle error: %v", err)
	}

	if hub.calls != 1 || hub.lastRoom != 7 {
		t.Fatalf("expected fanout to broadcast into hub, calls=%d room=%d", hub.calls, hub.lastRoom)
	}
}

func TestFanoutHandler_Handle_ContextCanceled(t *testing.T) {
	hub := &stubHub{
		codec: &stubCodec{encoded: []byte("ok")},
	}
	fanout, err := NewFanoutHandler[stubEvent](hub)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = fanout.Handle(ctx, stubEvent{Room: 99})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if hub.calls != 0 {
		t.Fatalf("expected no broadcast when context canceled, got %d", hub.calls)
	}
}
