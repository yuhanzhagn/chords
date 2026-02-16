package websocket

import (
	"connection/internal/service"
	kafkapb "connection/proto/kafka"
	"log"
	"sync"

	"google.golang.org/protobuf/proto"
)

type Client struct {
	ID         uint32
	Conn       *Connection
	SendChan   chan []byte
	msgService service.MessageService
}

type Room struct {
	ID      uint32
	Clients map[*Client]bool
	Lock    sync.Mutex
}

type Hub struct {
	store ConnectionStore
}

func NewHub(store ConnectionStore) *Hub {
	return &Hub{store: store}
}

func (h *Hub) AddClient(client *Client) {
	h.store.AddClient(client)
}

func (h *Hub) AddClientToRoom(clientID uint32, roomID uint32) {
	h.store.AddClientToRoom(clientID, roomID)
}

func (h *Hub) RemoveClient(clientID uint32) {
	h.store.RemoveClient(clientID)
}

func (h *Hub) RemoveClientFromRoom(clientID uint32, roomID uint32) {
	h.store.RemoveClientFromRoom(clientID, roomID)
}

func (h *Hub) Broadcast(roomID uint32, msg []byte) {
	clients := h.store.GetClientsInRoom(roomID)
	for _, c := range clients {
		select {
		case c.SendChan <- msg:
		default:
			// Handle full channel, e.g., drop message or disconnect client
		}
	}
}

func (h *Hub) HandleOutboundEvent(event *kafkapb.KafkaEvent) {
	// Unmarshal the outbound message to get roomID and body
	log.Printf("Hub handling outbound event: UserID=%d, RoomID=%d, MsgType=%s", event.UserId, event.RoomId, event.MsgType)
	rawbytes, err := proto.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal KafkaEvent: %v", err)
		return
	}
	h.Broadcast(event.RoomId, rawbytes)
}
