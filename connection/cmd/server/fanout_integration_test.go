package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connection/internal/source"
	kafkapb "connection/proto/kafka"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// This uses production WS mux/pipeline, HTTP source, protobuf codec and Hub.
// Backend authorization is mocked; Kafka/Redis and browser UI are not involved.
func TestAuthenticatedHTTPFanoutToWebSocket(t *testing.T) {
	const serviceToken = "isolated-fanout-test-credential-32-bytes"
	for _, scenario := range []string{"member", "anonymous push", "nonmember", "membership removed", "revoked session"} {
		t.Run(scenario, func(t *testing.T) {
			var revoked, nonmember atomic.Bool
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if revoked.Load() || r.Header.Get("Authorization") != "Bearer valid-access" {
					w.WriteHeader(401)
					return
				}
				var req struct {
					RoomID uint32 `json:"room_id"`
				}
				if json.NewDecoder(r.Body).Decode(&req) != nil {
					w.WriteHeader(400)
					return
				}
				if req.RoomID != 0 && (req.RoomID != 7 || nonmember.Load()) {
					w.WriteHeader(403)
					return
				}
				_ = json.NewEncoder(w).Encode(authGrant{UserID: 42, Username: "alice", SessionID: "session"})
			}))
			defer backend.Close()
			hub := newHub(newEventCodec("protobuf"))
			next := setupHandlerChain(hub, eventSinkFunc(func(_ context.Context, _ *kafkapb.KafkaEvent) error { t.Error("join must not reach sink"); return nil }), nil, "")
			wsServer := httptest.NewServer(newMux(hub, next, newBackendAuthorizer(backend.URL).authenticator(next)))
			defer wsServer.Close()
			fanoutSource, err := source.NewFanoutHTTPHandler(hub, ":0", serviceToken)
			require.NoError(t, err)
			mux := http.NewServeMux()
			mux.Handle("/fanout", fanoutSource)
			internal := httptest.NewServer(mux)
			defer internal.Close()
			push := func(token, content string) int {
				raw, err := json.Marshal(source.FanoutRequest{RoomID: 7, Event: &kafkapb.KafkaEvent{RoomId: 7, MsgType: "message", Content: []byte(content)}})
				require.NoError(t, err)
				req, err := http.NewRequest(http.MethodPost, internal.URL+"/fanout", bytes.NewReader(raw))
				require.NoError(t, err)
				if token != "" {
					req.Header.Set("X-Chords-Fanout-Token", token)
				}
				resp, err := internal.Client().Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()
				return resp.StatusCode
			}
			ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(wsServer.URL, "http")+"/ws", nil)
			require.NoError(t, err)
			defer ws.Close()
			require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": "valid-access"}))
			var ready map[string]string
			require.NoError(t, ws.ReadJSON(&ready))
			require.Equal(t, "ready", ready["type"])
			if scenario == "nonmember" {
				nonmember.Store(true)
			}
			raw, err := proto.Marshal(&kafkapb.KafkaEvent{RoomId: 7, MsgType: "join"})
			require.NoError(t, err)
			require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
			if scenario == "nonmember" {
				require.NoError(t, ws.SetReadDeadline(time.Now().Add(time.Second)))
				_, _, err = ws.ReadMessage()
				require.True(t, websocket.IsCloseError(err, 1008), "%v", err)
				require.False(t, hub.UserInGroup(42, 7))
				require.Equal(t, 204, push(serviceToken, "private"))
				return
			}
			require.Eventually(t, func() bool { return hub.UserInGroup(42, 7) }, time.Second, time.Millisecond)
			switch scenario {
			case "anonymous push":
				require.Equal(t, 401, push("", "forged-anonymous"))
				require.Equal(t, 401, push("valid-access", "forged-access-token"))
			case "membership removed":
				nonmember.Store(true)
			case "revoked session":
				revoked.Store(true)
			}
			require.Equal(t, 204, push(serviceToken, "private"))
			require.NoError(t, ws.SetReadDeadline(time.Now().Add(2*time.Second)))
			kind, payload, err := ws.ReadMessage()
			if scenario == "membership removed" || scenario == "revoked session" {
				require.True(t, websocket.IsCloseError(err, 1008), "protected message must not arrive: %q %v", payload, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, websocket.BinaryMessage, kind)
			var got kafkapb.KafkaEvent
			require.NoError(t, proto.Unmarshal(payload, &got))
			require.Equal(t, "private", string(got.Content), "an unauthorized push must never precede the legitimate event")
			require.Equal(t, uint32(7), got.RoomId)
		})
	}
}
