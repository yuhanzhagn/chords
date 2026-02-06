package websocket

import (
	"connection/internal/platform/kafka"
	"connection/internal/service"
	"encoding/json"
	"sync"
)

type Client struct {
	ID         uint
	Conn       *Connection
	SendChan   chan []byte
	msgService service.MessageService
}

type Room struct {
	ID      uint
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

func (h *Hub) AddClientToRoom(clientID uint, roomID uint) {
	h.store.AddClientToRoom(clientID, roomID)
}

func (h *Hub) RemoveClient(clientID uint) {
	h.store.RemoveClient(clientID)
}

func (h *Hub) RemoveClientFromRoom(clientID uint, roomID uint) {
	h.store.RemoveClientFromRoom(clientID, roomID)
}

func (h *Hub) Broadcast(roomID uint, msg []byte) {
	clients := h.store.GetClientsInRoom(roomID)
	for _, c := range clients {
		select {
		case c.SendChan <- msg:
		default:
			// Handle full channel, e.g., drop message or disconnect client
		}
	}
}

func (h *Hub) HandleOutboundEvent(event kafka.KafkaEvent) {
	// Unmarshal the outbound message to get roomID and body
	var outboundMsg struct {
		RoomID uint   `json:"room_id"`
		Body   []byte `json:"body"`
	}
	if err := json.Unmarshal(event.Payload, &outboundMsg); err != nil {
		return
	}

	h.Broadcast(outboundMsg.RoomID, outboundMsg.Body)
}
