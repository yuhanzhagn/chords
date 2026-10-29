package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connection/internal/gateway"
	"connection/internal/handler"
	kafkapb "connection/proto/kafka"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestWebSocketAuthenticationAndRevocation(t *testing.T) {
	var revoked atomic.Bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if revoked.Load() || r.Header.Get("Authorization") != "Bearer valid-token" {
			w.WriteHeader(401)
			return
		}
		var req struct {
			RoomID uint32 `json:"room_id"`
			Kind   string `json:"kind"`
			Digest string `json:"digest"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			w.WriteHeader(400)
			return
		}
		if req.RoomID != 0 && req.RoomID != 7 {
			w.WriteHeader(403)
			return
		}
		_ = json.NewEncoder(w).Encode(authGrant{UserID: 42, Username: "alice", SessionID: "session", CommandToken: "short-command-grant"})
	}))
	defer backend.Close()
	hub := newHub(newEventCodec("protobuf"))
	accepted := make(chan *kafkapb.KafkaEvent, 4)
	handlerFunc := func(c *handler.Context) error {
		inbound := c.Event.(gateway.InboundEvent[*kafkapb.KafkaEvent])
		if inbound.Event.MsgType == "join" {
			hub.AddClientToGroup(inbound.ClientID, inbound.Event.RoomId)
		}
		accepted <- inbound.Event
		return nil
	}
	server := httptest.NewServer(WsHandler(hub, handlerFunc, newBackendAuthorizer(backend.URL).legacyAuthenticator(handlerFunc)))
	defer server.Close()
	dial := func() *websocket.Conn {
		ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		require.NoError(t, err)
		t.Cleanup(func() { ws.Close() })
		return ws
	}
	send := func(ws *websocket.Conn, e *kafkapb.KafkaEvent) {
		raw, err := proto.Marshal(e)
		require.NoError(t, err)
		require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
	}
	assertRejected := func(ws *websocket.Conn) {
		require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
		_, _, err := ws.ReadMessage()
		require.True(t, websocket.IsCloseError(err, 1008), "%v", err)
	}
	for _, token := range []string{"", "invalid-token"} {
		ws := dial()
		require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": token}))
		assertRejected(ws)
	}
	anonymous := dial()
	send(anonymous, &kafkapb.KafkaEvent{MsgType: "join", UserId: 42, RoomId: 7})
	assertRejected(anonymous)
	require.Empty(t, hub.AllClients())
	require.Empty(t, accepted)
	authenticate := func() *websocket.Conn {
		ws := dial()
		require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": "valid-token"}))
		var ready map[string]string
		require.NoError(t, ws.ReadJSON(&ready))
		require.Equal(t, "ready", ready["type"])
		return ws
	}
	ws := authenticate()
	send(ws, &kafkapb.KafkaEvent{MsgType: "join", UserId: 42, RoomId: 7})
	select {
	case event := <-accepted:
		require.Equal(t, uint32(42), event.UserId)
	case <-time.After(time.Second):
		t.Fatal("join not accepted")
	}
	// The authenticated identity cannot be changed by a subsequent event.
	send(ws, &kafkapb.KafkaEvent{MsgType: "message", UserId: 99, RoomId: 7})
	assertRejected(ws)
	require.Empty(t, accepted)
	ws = authenticate()
	send(ws, &kafkapb.KafkaEvent{MsgType: "join", UserId: 42, RoomId: 8})
	assertRejected(ws)
	require.Empty(t, accepted)
	ws = authenticate()
	send(ws, &kafkapb.KafkaEvent{MsgType: "join", RoomId: 7})
	select {
	case event := <-accepted:
		require.Equal(t, uint32(42), event.UserId)
	case <-time.After(time.Second):
		t.Fatal("join not accepted")
	}
	// Live validation also prevents an idle revoked subscriber receiving a broadcast.
	revoked.Store(true)
	hub.Broadcast(7, []byte("private message"))
	assertRejected(ws)
	require.Empty(t, accepted)
	ws = dial()
	require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": "valid-token"}))
	assertRejected(ws)
}

func TestAuthenticatedHandlerFailsClosed(t *testing.T) {
	for _, scenario := range []string{"offline", "principal changed", "revoked"} {
		t.Run(scenario, func(t *testing.T) {
			var requests atomic.Int32
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := requests.Add(1)
				if n > 1 && scenario == "revoked" {
					w.WriteHeader(401)
					return
				}
				uid := uint32(42)
				if n > 1 && scenario == "principal changed" {
					uid = 99
				}
				_ = json.NewEncoder(w).Encode(authGrant{UserID: uid, Username: "alice", SessionID: "session", CommandToken: "grant"})
			}))
			defer backend.Close()
			called := false
			auth, err := newBackendAuthorizer(backend.URL).legacyAuthenticator(func(*handler.Context) error { called = true; return nil })(context.Background(), []byte(`{"type":"auth","token":"valid"}`))
			require.NoError(t, err)
			if scenario == "offline" {
				backend.Close()
			}
			err = auth.Handler(&handler.Context{Context: context.Background(), ClientID: 1, Values: map[string]any{handler.RequestContextKey: httptest.NewRequest(http.MethodGet, "/ws", nil)}, Event: gateway.InboundEvent[*kafkapb.KafkaEvent]{Event: &kafkapb.KafkaEvent{UserId: 42, RoomId: 7, MsgType: "message"}}})
			if scenario == "offline" {
				require.ErrorIs(t, err, gateway.ErrAuthUnavailable)
			} else {
				require.ErrorIs(t, err, gateway.ErrPolicyViolation)
			}
			require.False(t, called)
		})
	}
}

func TestWebSocketRequiresTimelyFirstFrame(t *testing.T) {
	hub := newHub(newEventCodec("protobuf"))
	var called atomic.Bool
	handlerFunc := func(*handler.Context) error { return nil }
	server := httptest.NewServer(WsHandler(hub, handlerFunc, func(context.Context, []byte) (gateway.AuthenticatedSession, error) {
		called.Store(true)
		return gateway.AuthenticatedSession{}, nil
	}))
	defer server.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	defer ws.Close()
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(7*time.Second)))
	_, _, err = ws.ReadMessage()
	require.True(t, websocket.IsCloseError(err, 1008), "%v", err)
	require.False(t, called.Load())
	require.Empty(t, hub.AllClients())
}

type eventSinkFunc func(context.Context, *kafkapb.KafkaEvent) error

func (f eventSinkFunc) Write(ctx context.Context, e *kafkapb.KafkaEvent) error { return f(ctx, e) }
func TestAuthenticatedMessageUsesProductionPipeline(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(authGrant{UserID: 42, Username: "alice", SessionID: "session", CommandToken: "grant"})
	}))
	defer backend.Close()
	hub := newHub(newEventCodec("protobuf"))
	messages := make(chan *kafkapb.KafkaEvent, 1)
	next := setupHandlerChain(hub, eventSinkFunc(func(_ context.Context, e *kafkapb.KafkaEvent) error { messages <- e; return nil }), nil, "")
	server := httptest.NewServer(newMux(hub, next, newBackendAuthorizer(backend.URL).legacyAuthenticator(next)))
	defer server.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", nil)
	require.NoError(t, err)
	defer ws.Close()
	require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": "valid"}))
	var ready map[string]string
	require.NoError(t, ws.ReadJSON(&ready))
	require.Equal(t, "ready", ready["type"])
	for _, kind := range []string{"join", "message"} {
		raw, err := proto.Marshal(&kafkapb.KafkaEvent{RoomId: 7, MsgType: kind, Content: []byte("hello")})
		require.NoError(t, err)
		require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
	}
	select {
	case e := <-messages:
		require.Equal(t, uint32(42), e.UserId)
		require.Equal(t, uint32(7), e.RoomId)
		require.Equal(t, "message", e.MsgType)
	case <-time.After(time.Second):
		t.Fatal("authorized message did not reach sink")
	}
	require.True(t, hub.UserInGroup(42, 7))
	backend.Close()
	raw, err := proto.Marshal(&kafkapb.KafkaEvent{RoomId: 7, MsgType: "message"})
	require.NoError(t, err)
	require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(time.Second)))
	_, _, err = ws.ReadMessage()
	require.True(t, websocket.IsCloseError(err, 1013), "%v", err)
}

func TestSendAdmissionBeforeQueueAndSignatureCoversNewFields(t *testing.T) {
	var messageAuthorizations atomic.Int32
	var digest atomic.Value
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Kind   string `json:"kind"`
			Digest string `json:"digest"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
		if in.Kind == "message" {
			messageAuthorizations.Add(1)
			digest.Store(in.Digest)
		}
		_ = json.NewEncoder(w).Encode(authGrant{UserID: 42, Username: "alice", SessionID: "session", CommandToken: "grant"})
	}))
	defer backend.Close()
	hub := newHub(newEventCodec("protobuf"))
	messages := make(chan *kafkapb.KafkaEvent, 1)
	next := setupHandlerChain(hub, eventSinkFunc(func(_ context.Context, e *kafkapb.KafkaEvent) error { messages <- e; return nil }), nil, "")
	server := httptest.NewServer(newMux(hub, next, newBackendAuthorizer(backend.URL).legacyAuthenticator(next)))
	defer server.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws", nil)
	require.NoError(t, err)
	defer ws.Close()
	require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": "valid"}))
	var ready map[string]string
	require.NoError(t, ws.ReadJSON(&ready))
	input := &kafkapb.KafkaEvent{MsgType: "message", RoomId: 7, Content: []byte(strings.Repeat("x", kafkapb.MaxContentBytes)), RequestId: strings.Repeat("x", 128), TempId: strings.Repeat("\\", 128)}
	raw, err := proto.Marshal(input)
	require.NoError(t, err)
	require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
	select {
	case event := <-messages:
		actual, err := proto.Marshal(event)
		require.NoError(t, err)
		hash := sha256.Sum256(actual)
		require.Equal(t, hex.EncodeToString(hash[:]), digest.Load())
		require.Equal(t, input.RequestId, event.RequestId)
	case <-time.After(2 * time.Second):
		t.Fatal("boundary message rejected")
	}
	input.Content = append(input.Content, 'x')
	raw, err = proto.Marshal(input)
	require.NoError(t, err)
	require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
	_, _, err = ws.ReadMessage()
	require.True(t, websocket.IsCloseError(err, 1009), "%v", err)
	require.EqualValues(t, 1, messageAuthorizations.Load(), "oversize input must be rejected before command authorization and queueing")
	require.Empty(t, messages)
}
