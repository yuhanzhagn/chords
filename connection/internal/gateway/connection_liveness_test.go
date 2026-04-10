package gateway

import (
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connection/internal/handler"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWSConn struct {
	mu sync.Mutex

	readLimit      int64
	readLimitCalls int

	readDeadlines  []time.Time
	writeDeadlines []time.Time

	pongHandler func(string) error

	readMessageFunc  func() (int, []byte, error)
	writeMessageFunc func(int, []byte) error

	closeCalls int
}

func (m *mockWSConn) ReadMessage() (int, []byte, error) {
	if m.readMessageFunc != nil {
		return m.readMessageFunc()
	}
	return 0, nil, errors.New("read message not mocked")
}

func (m *mockWSConn) WriteMessage(messageType int, data []byte) error {
	if m.writeMessageFunc != nil {
		return m.writeMessageFunc(messageType, data)
	}
	return nil
}

func (m *mockWSConn) Close() error {
	m.mu.Lock()
	m.closeCalls++
	m.mu.Unlock()
	return nil
}

func (m *mockWSConn) SetWriteDeadline(t time.Time) error {
	m.mu.Lock()
	m.writeDeadlines = append(m.writeDeadlines, t)
	m.mu.Unlock()
	return nil
}

func (m *mockWSConn) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	m.readDeadlines = append(m.readDeadlines, t)
	m.mu.Unlock()
	return nil
}

func (m *mockWSConn) SetReadLimit(limit int64) {
	m.mu.Lock()
	m.readLimit = limit
	m.readLimitCalls++
	m.mu.Unlock()
}

func (m *mockWSConn) SetPongHandler(h func(string) error) {
	m.mu.Lock()
	m.pongHandler = h
	m.mu.Unlock()
}

func TestReadPump_SetsReadLimitAndPongHandler(t *testing.T) {
	// Arrange
	prevMax := maxMessageSize
	prevPong := pongWait
	maxMessageSize = 123
	pongWait = 50 * time.Millisecond
	defer func() {
		maxMessageSize = prevMax
		pongWait = prevPong
	}()

	ws := &mockWSConn{
		readMessageFunc: func() (int, []byte, error) {
			return 0, nil, errors.New("boom")
		},
	}
	conn, err := NewConnection(ws, func(*handler.Context) error { return nil }, httptest.NewRequest("GET", "/ws", nil))
	require.NoError(t, err)
	client := &Client{ID: 1, Conn: conn, SendChan: make(chan []byte, 1)}

	// Act
	readPump[struct{}](client, nil)

	// Assert
	ws.mu.Lock()
	readLimit := ws.readLimit
	readLimitCalls := ws.readLimitCalls
	pongHandler := ws.pongHandler
	readDeadlineCalls := len(ws.readDeadlines)
	ws.mu.Unlock()

	assert.Equal(t, int64(123), readLimit)
	assert.Equal(t, 1, readLimitCalls)
	assert.NotNil(t, pongHandler)
	assert.GreaterOrEqual(t, readDeadlineCalls, 1)

	before := readDeadlineCalls
	err = pongHandler("pong")
	require.NoError(t, err)
	ws.mu.Lock()
	after := len(ws.readDeadlines)
	ws.mu.Unlock()
	assert.Equal(t, before+1, after)
}

func TestWritePump_SendsPing(t *testing.T) {
	// Arrange
	prevPing := pingPeriod
	prevWrite := writeWait
	pingPeriod = 5 * time.Millisecond
	writeWait = 5 * time.Millisecond
	defer func() {
		pingPeriod = prevPing
		writeWait = prevWrite
	}()

	pingCh := make(chan struct{}, 1)
	ws := &mockWSConn{
		writeMessageFunc: func(messageType int, data []byte) error {
			if messageType == websocket.PingMessage {
				pingCh <- struct{}{}
				return errors.New("stop")
			}
			return nil
		},
	}
	conn, err := NewConnection(ws, func(*handler.Context) error { return nil }, httptest.NewRequest("GET", "/ws", nil))
	require.NoError(t, err)
	client := &Client{ID: 1, Conn: conn, SendChan: make(chan []byte, 1), wsMsgType: websocket.BinaryMessage}

	done := make(chan struct{})

	// Act
	go func() {
		writePump(client)
		close(done)
	}()

	// Assert
	select {
	case <-pingCh:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected ping to be sent")
	}

	select {
	case <-done:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected writePump to stop after ping error")
	}
}
