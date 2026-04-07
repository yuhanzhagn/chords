package source

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connection/internal/event/codec"
	"connection/internal/gateway"
	kafkapb "connection/proto/kafka"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockEventCodec struct {
	mock.Mock
}

func (m *mockEventCodec) Encode(event *kafkapb.KafkaEvent) ([]byte, error) {
	args := m.Called(event)
	var payload []byte
	if raw := args.Get(0); raw != nil {
		payload = raw.([]byte)
	}
	return payload, args.Error(1)
}

func (m *mockEventCodec) Decode(data []byte) (*kafkapb.KafkaEvent, error) {
	args := m.Called(data)
	var event *kafkapb.KafkaEvent
	if raw := args.Get(0); raw != nil {
		event = raw.(*kafkapb.KafkaEvent)
	}
	return event, args.Error(1)
}

var _ codec.EventCodec[*kafkapb.KafkaEvent] = (*mockEventCodec)(nil)

func newTestHub(t *testing.T, eventCodec codec.EventCodec[*kafkapb.KafkaEvent]) *gateway.Hub[*kafkapb.KafkaEvent] {
	t.Helper()
	router := gateway.EventRouter[*kafkapb.KafkaEvent]{
		MsgType: func(e *kafkapb.KafkaEvent) string {
			if e == nil {
				return ""
			}
			return e.MsgType
		},
		GroupID: func(e *kafkapb.KafkaEvent) uint32 {
			if e == nil {
				return 0
			}
			return e.RoomId
		},
	}
	return gateway.NewHub(gateway.NewMemoryStore(), eventCodec, router)
}

func newClient(id uint32) *gateway.Client {
	return &gateway.Client{ID: id, SendChan: make(chan []byte, 1)}
}

func TestFanoutHTTPSource_ServeHTTP_MethodNotAllowed(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	hub := newTestHub(t, codecMock)
	source := NewFanoutHTTPHandler(hub, ":0")
	req := httptest.NewRequest(http.MethodGet, "/fanout", nil)
	recorder := httptest.NewRecorder()

	// Act
	source.ServeHTTP(recorder, req)

	// Assert
	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	assert.Equal(t, http.MethodPost, recorder.Header().Get("Allow"))
	assert.Contains(t, recorder.Body.String(), "method not allowed")
}

func TestFanoutHTTPSource_ServeHTTP_InvalidJSON(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	hub := newTestHub(t, codecMock)
	source := NewFanoutHTTPHandler(hub, ":0")
	req := httptest.NewRequest(http.MethodPost, "/fanout", bytes.NewBufferString("{"))
	recorder := httptest.NewRecorder()

	// Act
	source.ServeHTTP(recorder, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "invalid json")
}

func TestFanoutHTTPSource_ServeHTTP_MissingEvent(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	hub := newTestHub(t, codecMock)
	source := NewFanoutHTTPHandler(hub, ":0")
	payload, err := json.Marshal(FanoutRequest{RoomID: 1, UserIDs: []uint32{2}})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/fanout", bytes.NewBuffer(payload))
	recorder := httptest.NewRecorder()

	// Act
	source.ServeHTTP(recorder, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "event is required")
}

func TestFanoutHTTPSource_ServeHTTP_BroadcastsToRoom(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	codecMock.On("Encode", mock.Anything).Return([]byte("encoded"), nil)
	hub := newTestHub(t, codecMock)
	client := newClient(10)
	hub.AddClient(client)
	hub.AddClientToGroup(client.ID, 7)
	source := NewFanoutHTTPHandler(hub, ":0")

	event := &kafkapb.KafkaEvent{RoomId: 7, MsgType: "message"}
	payload, err := json.Marshal(FanoutRequest{RoomID: 7, Event: event})
	require.NoError(t, err)
	q := httptest.NewRequest(http.MethodPost, "/fanout", bytes.NewBuffer(payload))
	recorder := httptest.NewRecorder()

	// Act
	source.ServeHTTP(recorder, q)

	// Assert
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	select {
	case got := <-client.SendChan:
		assert.Equal(t, []byte("encoded"), got)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected payload to be broadcast to client")
	}
	codecMock.AssertNumberOfCalls(t, "Encode", 1)
}

func TestFanoutHTTPSource_ServeHTTP_SendsToUsers(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	codecMock.On("Encode", mock.Anything).Return([]byte("encoded"), nil)
	hub := newTestHub(t, codecMock)
	clientA := newClient(100)
	clientB := newClient(200)
	hub.AddClient(clientA)
	hub.AddClient(clientB)
	source := NewFanoutHTTPHandler(hub, ":0")

	event := &kafkapb.KafkaEvent{RoomId: 0, MsgType: "message"}
	payload, err := json.Marshal(FanoutRequest{UserIDs: []uint32{100, 200}, Event: event})
	require.NoError(t, err)
	q := httptest.NewRequest(http.MethodPost, "/fanout", bytes.NewBuffer(payload))
	recorder := httptest.NewRecorder()

	// Act
	source.ServeHTTP(recorder, q)

	// Assert
	assert.Equal(t, http.StatusNoContent, recorder.Code)
	select {
	case got := <-clientA.SendChan:
		assert.Equal(t, []byte("encoded"), got)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected payload to be sent to client A")
	}
	select {
	case got := <-clientB.SendChan:
		assert.Equal(t, []byte("encoded"), got)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected payload to be sent to client B")
	}
	codecMock.AssertNumberOfCalls(t, "Encode", 1)
}

func TestApplyFanout_Errors(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	codecMock.On("Encode", mock.Anything).Return([]byte("encoded"), nil)
	hub := newTestHub(t, codecMock)

	// Act + Assert
	err := applyFanout(nil, &FanoutRequest{Event: &kafkapb.KafkaEvent{}})
	assert.EqualError(t, err, "hub is required")

	err = applyFanout(hub, nil)
	assert.EqualError(t, err, "event is required")

	err = applyFanout(hub, &FanoutRequest{})
	assert.EqualError(t, err, "event is required")

	err = applyFanout(hub, &FanoutRequest{Event: &kafkapb.KafkaEvent{}})
	assert.EqualError(t, err, "room_id or user_ids is required")
	codecMock.AssertNumberOfCalls(t, "Encode", 1)
}

func TestApplyFanout_EncodeError(t *testing.T) {
	// Arrange
	codecMock := &mockEventCodec{}
	codecMock.On("Encode", mock.Anything).Return(nil, errors.New("boom"))
	hub := newTestHub(t, codecMock)
	request := &FanoutRequest{RoomID: 7, Event: &kafkapb.KafkaEvent{RoomId: 7, MsgType: "message"}}

	// Act
	err := applyFanout(hub, request)

	// Assert
	assert.EqualError(t, err, "failed to encode event")
	codecMock.AssertNumberOfCalls(t, "Encode", 1)
}
