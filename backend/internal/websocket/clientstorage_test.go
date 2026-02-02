package websocket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetGetRemoveClient(t *testing.T) {
	c := &Client{ID: 42, SendChan: make(chan []byte, 1)}

	// Add client
	setClient(c)

	// Retrieve client
	got := getClient(42)
	require.NotNil(t, got)
	require.Equal(t, c.ID, got.ID)

	// Remove client
	removeClient(42)
	got = getClient(42)
	require.Nil(t, got)
}
