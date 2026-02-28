package gateway

import (
	"connection/internal/event/codec"
	"log"

	gws "github.com/gorilla/websocket"
)

type Client struct {
	ID        uint32
	Conn      *Connection
	SendChan  chan []byte
	wsMsgType int
}

type EventRouter[T any] struct {
	MsgType     func(T) string
	GroupID     func(T) uint32
	JoinType    string
	LeaveType   string
	MessageType string
}

func (r EventRouter[T]) withDefaults() EventRouter[T] {
	if r.JoinType == "" {
		r.JoinType = "join"
	}
	if r.LeaveType == "" {
		r.LeaveType = "leave"
	}
	if r.MessageType == "" {
		r.MessageType = "message"
	}
	return r
}

type Hub[T any] struct {
	store ConnectionStore
	codec codec.EventCodec[T]
	event EventRouter[T]
}

func NewHub[T any](store ConnectionStore, eventCodec codec.EventCodec[T], router EventRouter[T]) *Hub[T] {
	if eventCodec == nil {
		panic("event codec is required")
	}
	if router.MsgType == nil || router.GroupID == nil {
		panic("event router MsgType and GroupID are required")
	}
	return &Hub[T]{
		store: store,
		codec: eventCodec,
		event: router.withDefaults(),
	}
}

func (h *Hub[T]) AddClient(client *Client) {
	h.store.AddClient(client)
}

func (h *Hub[T]) AddClientToGroup(clientID uint32, groupID uint32) {
	h.store.AssignClientToGroup(clientID, groupID)
}

func (h *Hub[T]) RemoveClient(clientID uint32) {
	h.store.RemoveClient(clientID)
}

func (h *Hub[T]) RemoveClientFromGroup(clientID uint32, groupID uint32) {
	h.store.RemoveClientFromGroup(clientID, groupID)
}

func (h *Hub[T]) Broadcast(groupID uint32, msg []byte) {
	clients := h.store.GetClientsInGroup(groupID)
	for _, c := range clients {
		select {
		case c.SendChan <- msg:
		default:
			// Handle full channel, e.g., drop message or disconnect client
		}
	}
}

func (h *Hub[T]) Codec() codec.EventCodec[T] {
	return h.codec
}

func (h *Hub[T]) MsgType(event T) string {
	return h.event.MsgType(event)
}

func (h *Hub[T]) GroupID(event T) uint32 {
	return h.event.GroupID(event)
}

func (h *Hub[T]) IsJoin(event T) bool {
	return h.MsgType(event) == h.event.JoinType
}

func (h *Hub[T]) IsLeave(event T) bool {
	return h.MsgType(event) == h.event.LeaveType
}

func (h *Hub[T]) IsMessage(event T) bool {
	return h.MsgType(event) == h.event.MessageType
}

func (h *Hub[T]) WSMessageType() int {
	if provider, ok := h.codec.(codec.WSMessageTypeProvider); ok {
		return provider.WSMessageType()
	}
	return gws.BinaryMessage
}

func (h *Hub[T]) HandleOutboundEvent(event T) {
	log.Printf("Hub handling outbound event: GroupID=%d, MsgType=%s", h.GroupID(event), h.MsgType(event))
	rawbytes, err := h.codec.Encode(event)
	if err != nil {
		log.Printf("Failed to encode outbound event: %v", err)
		return
	}
	h.Broadcast(h.GroupID(event), rawbytes)
}
