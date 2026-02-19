package gateway

import (
	"connection/internal/event/codec"
	"connection/internal/service"
	"log"
	"sync"

	gws "github.com/gorilla/websocket"
)

type Client struct {
	ID         uint32
	Conn       *Connection
	SendChan   chan []byte
	wsMsgType  int
	msgService service.MessageService
}

type Room struct {
	ID      uint32
	Clients map[*Client]bool
	Lock    sync.Mutex
}

type EventRouter[T any] struct {
	MsgType     func(T) string
	RoomID      func(T) uint32
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
	if router.MsgType == nil || router.RoomID == nil {
		panic("event router MsgType and RoomID are required")
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

func (h *Hub[T]) AddClientToRoom(clientID uint32, roomID uint32) {
	h.store.AssignClientToGroup(clientID, roomID)
}

func (h *Hub[T]) RemoveClient(clientID uint32) {
	h.store.RemoveClient(clientID)
}

func (h *Hub[T]) RemoveClientFromRoom(clientID uint32, roomID uint32) {
	h.store.RemoveClientFromGroup(clientID, roomID)
}

func (h *Hub[T]) Broadcast(roomID uint32, msg []byte) {
	clients := h.store.GetClientsInGroup(roomID)
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

func (h *Hub[T]) RoomID(event T) uint32 {
	return h.event.RoomID(event)
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
	log.Printf("Hub handling outbound event: RoomID=%d, MsgType=%s", h.RoomID(event), h.MsgType(event))
	rawbytes, err := h.codec.Encode(event)
	if err != nil {
		log.Printf("Failed to encode outbound event: %v", err)
		return
	}
	h.Broadcast(h.RoomID(event), rawbytes)
}
