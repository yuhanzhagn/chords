package websocket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHubAddRemoveBroadcast(t *testing.T) {
	hub := &Hub{Rooms: make(map[uint]*Room)}

	// Create mock clients
	c1 := &Client{ID: 1, SendChan: make(chan []byte, 1)}
	c2 := &Client{ID: 2, SendChan: make(chan []byte, 1)}

	// Add clients to room 100
	hub.AddClient(100, c1)
	hub.AddClient(100, c2)

	room, ok := hub.Rooms[100]
	require.True(t, ok)
	require.Len(t, room.Clients, 2)

	// Broadcast a message
	msg := []byte("hello")
	hub.Broadcast(100, msg)

	// Clients should receive message
	require.Equal(t, msg, <-c1.SendChan)
	require.Equal(t, msg, <-c2.SendChan)

	// Remove a client
	hub.RemoveClient(100, c1)
	require.Len(t, room.Clients, 1)
	require.Contains(t, room.Clients, c2)
}
