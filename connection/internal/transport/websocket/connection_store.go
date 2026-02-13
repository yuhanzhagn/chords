// websocket/store.go
package websocket

import "sync"

type ConnectionStore interface {
	AddClient(c *Client)
	RemoveClient(clientID uint32)

	AddClientToRoom(clientID uint32, roomID uint32)
	RemoveClientFromRoom(clientID uint32, roomID uint32)

	GetClient(clientID uint32) *Client
	GetClientsInRoom(roomID uint32) []*Client
	GetAllClients() []*Client
}

type MemoryStore struct {
	mu      sync.RWMutex
	clients map[uint32]*Client
	rooms   map[uint32]map[uint32]*Client // roomID -> clientID -> client
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		clients: make(map[uint32]*Client),
		rooms:   make(map[uint32]map[uint32]*Client),
	}
}

func (s *MemoryStore) AddClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
}

func (s *MemoryStore) RemoveClient(clientID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, clientID)
	for _, room := range s.rooms {
		delete(room, clientID)
	}
}

func (s *MemoryStore) GetClient(clientID uint32) *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[clientID]
}

func (s *MemoryStore) AddClientToRoom(clientID uint32, roomID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.rooms[roomID]; !ok {
		s.rooms[roomID] = make(map[uint32]*Client)
	}

	if c, ok := s.clients[clientID]; ok {
		s.rooms[roomID][clientID] = c
	}
}

func (s *MemoryStore) RemoveClientFromRoom(clientID uint32, roomID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, ok := s.rooms[roomID]; ok {
		delete(room, clientID)
	}
}

func (s *MemoryStore) GetClientsInRoom(roomID uint32) []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []*Client
	for _, c := range s.rooms[roomID] {
		res = append(res, c)
	}
	return res
}

func (s *MemoryStore) GetAllClients() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []*Client
	for _, c := range s.clients {
		res = append(res, c)
	}
	return res
}
