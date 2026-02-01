package websocket

import (
    "fmt"
    "sync"
	"backend/internal/service"
)

type Client struct {
    ID       uint
    Conn     *Connection
    SendChan chan []byte
	msgService service.MessageService
}

type Room struct {
    ID      uint
    Clients map[*Client]bool
    Lock    sync.Mutex
}

type Hub struct {
    Rooms map[uint]*Room
    Lock  sync.Mutex
}

var GlobalHub = &Hub{
    Rooms: make(map[uint]*Room),
}

// Add client to a room
func (h *Hub) AddClient(roomID uint, client *Client) {
    h.Lock.Lock()
    room, exists := h.Rooms[roomID]
    if !exists {
        room = &Room{
            ID:      roomID,
            Clients: make(map[*Client]bool),
        }
        h.Rooms[roomID] = room
    }
    h.Lock.Unlock()

    room.Lock.Lock()
    room.Clients[client] = true
    room.Lock.Unlock()

    fmt.Printf("Client %d joined room %d\n", client.ID, roomID)
}

// Remove client
func (h *Hub) RemoveClient(roomID uint, client *Client) {
    if room, ok := h.Rooms[roomID]; ok {
        room.Lock.Lock()
        delete(room.Clients, client)
        room.Lock.Unlock()
        fmt.Printf("Client %d left room %d\n", client.ID, roomID)
    }
}

// Broadcast message to all clients in room
func (h *Hub) Broadcast(roomID uint, message []byte) {
    if room, ok := h.Rooms[roomID]; ok {
        room.Lock.Lock()
        for c := range room.Clients {
            select {
            case c.SendChan <- message:
            default:
                // if channel blocked, remove client
                delete(room.Clients, c)
            }
        }
        room.Lock.Unlock()
    }
}

