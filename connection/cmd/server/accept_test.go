package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connection/internal/handler"
	kafkapb "connection/proto/kafka"
	"github.com/stretchr/testify/require"
)

func TestPersistedReplyRequiresCompleteCorrelatedReceipt(t *testing.T) {
	for _, fault := range []string{"valid", "wrong-request", "wrong-room", "wrong-user", "missing-seq", "bad-time", "empty", "redirect", "unauthorized", "forbidden", "conflict", "oversize", "unavailable"} {
		t.Run(fault, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer isolated", r.Header.Get("Authorization"))
				require.Equal(t, "/api/messages", r.URL.Path)
				receipt := map[string]any{"UserID": uint32(1), "RoomID": uint32(7), "request_id": "request", "event_id": "event", "server_message_id": "9007199254740993", "persisted_at": "2026-01-01T00:00:00Z", "room_seq": "9007199254740993"}
				switch fault {
				case "wrong-request":
					receipt["request_id"] = "other"
				case "wrong-room":
					receipt["RoomID"] = 8
				case "wrong-user":
					receipt["UserID"] = 2
				case "missing-seq":
					delete(receipt, "room_seq")
				case "bad-time":
					receipt["persisted_at"] = "invalid"
				case "empty":
					w.WriteHeader(204)
					return
				case "redirect":
					w.Header().Set("Location", "/must-not-follow")
					w.WriteHeader(307)
					return
				case "unauthorized":
					w.WriteHeader(401)
					return
				case "forbidden":
					w.WriteHeader(403)
					return
				case "conflict":
					w.WriteHeader(409)
					return
				case "oversize":
					w.WriteHeader(413)
					return
				case "unavailable":
					w.WriteHeader(503)
					return
				}
				_ = json.NewEncoder(w).Encode(receipt)
			}))
			defer backend.Close()
			var reply *kafkapb.KafkaEvent
			c := &handler.Context{Context: context.Background(), Reply: func(v any) error { reply = v.(*kafkapb.KafkaEvent); return nil }}
			require.NoError(t, newBackendAuthorizer(backend.URL).accept(c, "isolated", &kafkapb.KafkaEvent{UserId: 1, RoomId: 7, RequestId: "request", Content: []byte("hello")}))
			require.NotNil(t, reply)
			if fault == "valid" {
				require.Equal(t, "persisted", reply.MsgType)
				require.EqualValues(t, 9007199254740993, reply.RoomSeq)
				return
			}
			require.Equal(t, "send_error", reply.MsgType)
			if fault == "unauthorized" || fault == "forbidden" || fault == "conflict" || fault == "oversize" {
				require.Contains(t, string(reply.Content), `"outcome":"rejected"`)
			} else {
				require.Contains(t, string(reply.Content), `"outcome":"unknown"`)
			}
		})
	}
}
