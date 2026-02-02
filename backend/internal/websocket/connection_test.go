package websocket

import (
	"testing"
	"time"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
//	"bytes"
	"encoding/json"
)

// Mock connection that implements Ws field with Write/ReadMessage
type MockConn struct {
	Messages [][]byte
	ReadIdx  int
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
    return nil // no-op for testing
}

func (m *MockConn) ReadMessage() (int, []byte, error) {
	if m.ReadIdx >= len(m.Messages) {
		return 0, nil, websocket.ErrCloseSent
	}
	msg := m.Messages[m.ReadIdx]
	m.ReadIdx++
	return websocket.TextMessage, msg, nil
}

func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	m.Messages = append(m.Messages, data)
	return nil
}

func (m *MockConn) Close() error {
	return nil
}

func TestReadAndParseWSMessage(t *testing.T) {
	wsMsg := WSMessage{
		MsgType: "SUBSCRIBE",
		UserID:  1,
		RoomID:  100,
		Message: "hello",
	}

	msgBytes, _ := json.Marshal(wsMsg)
	mockConn := &MockConn{Messages: [][]byte{msgBytes}}

	c := &Client{
		Conn: &Connection{
			Ws:   mockConn,
			Send: make(chan []byte, 1),
		},
	}

	got := readAndParseWSMessage(c)
	require.NotNil(t, got)
	require.Equal(t, wsMsg.MsgType, got.MsgType)
	require.Equal(t, wsMsg.UserID, got.UserID)
	require.Equal(t, wsMsg.RoomID, got.RoomID)
	require.Equal(t, wsMsg.Message, got.Message)
}
