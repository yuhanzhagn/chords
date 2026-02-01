package websocket

import (
	"log"
	"sync"
)

var clientMap = struct {
	sync.RWMutex
	clients map[uint]*Client
}{clients: make(map[uint]*Client)}

// Add or update a client
func setClient(c *Client) {
	clientMap.Lock()
	defer clientMap.Unlock()
	clientMap.clients[c.ID] = c
	log.Printf("Client %d added/updated\n", c.ID)
}

// Get a client by ID
func getClient(id uint) *Client {
	clientMap.RLock()
	defer clientMap.RUnlock()
	if c, ok := clientMap.clients[id]; ok {
		return c
	}
	return nil
}

func removeClient(id uint) {
    clientMap.Lock()
    defer clientMap.Unlock()
    if _, ok := clientMap.clients[id]; ok {
        delete(clientMap.clients, id)
        log.Printf("Client %d removed\n", id)
    } else {
        log.Printf("Client %d not found\n", id)
    }
}
