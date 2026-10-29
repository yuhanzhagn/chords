package controller_service_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Same logical request after a lost response must return the committed receipt.
func TestRESTSendRequestReplay(t *testing.T) {
	r := setupBasicMiddleware()
	setupMessageRoute(r)
	send := func(content string) *httptest.ResponseRecorder {
		b, err := json.Marshal(map[string]any{"user_id": 1, "chat_room_id": 1, "content": content, "request_id": "lost-response"})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	first, retry := send("once"), send("once")
	require.Equal(t, 201, first.Code)
	require.Equal(t, 200, retry.Code)
	var a, b map[string]any
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &a))
	require.NoError(t, json.Unmarshal(retry.Body.Bytes(), &b))
	for _, key := range []string{"server_message_id", "event_id", "persisted_at", "request_id"} {
		require.NotEmpty(t, a[key])
		require.Equal(t, a[key], b[key])
	}
	require.Equal(t, 409, send("different").Code)
	req := httptest.NewRequest("DELETE", "/api/messages/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)
	deleted := send("once")
	require.Equal(t, 200, deleted.Code)
	require.Contains(t, deleted.Body.String(), `"deleted":true`)
}

func TestRESTSendUTF8AndEscapedSizeBoundaries(t *testing.T) {
	r := setupBasicMiddleware()
	setupMessageRoute(r)
	for i, content := range []string{strings.Repeat("\x00", 256*1024), strings.Repeat("界", (256*1024)/3), strings.Repeat("a", 256*1024+1), strings.Repeat("界", (256*1024)/3+1)} {
		raw, err := json.Marshal(map[string]any{"user_id": 1, "chat_room_id": 1, "content": content, "request_id": fmt.Sprint(i)})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if i < 2 {
			require.Equal(t, 201, w.Code)
		} else {
			require.Equal(t, 413, w.Code)
		}
	}
	// Even a valid short object with an oversized trailing body must be rejected explicitly.
	raw := []byte(`{"user_id":1,"chat_room_id":1,"content":"small"}`)
	raw = append(raw, bytes.Repeat([]byte(" "), 6*256*1024+4097)...)
	req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 413, w.Code)
}

func TestRESTIdentifierAdmission(t *testing.T) {
	r := setupBasicMiddleware()
	setupMessageRoute(r)
	for _, tc := range []struct {
		id     string
		status int
	}{{strings.Repeat("<", 128), 201}, {strings.Repeat("x", 129), 413}, {"with space", 400}} {
		raw, err := json.Marshal(map[string]any{"user_id": 1, "chat_room_id": 1, "content": "hello", "request_id": tc.id, "temp_id": strings.Repeat("<", 128)})
		require.NoError(t, err)
		req := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, tc.status, w.Code)
	}
}

func TestRESTSequenceCursorValidationAndStringReceipts(t *testing.T) {
	r := setupBasicMiddleware()
	setupMessageRoute(r)
	raw := []byte(`{"user_id":1,"chat_room_id":1,"content":"sequence","request_id":"seq-http"}`)
	request := httptest.NewRequest("POST", "/api/messages", bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	require.Equal(t, 201, response.Code)
	require.Contains(t, response.Body.String(), `"room_seq":"1"`)
	for _, tc := range []struct {
		query  string
		status int
	}{
		{"after_seq=0&limit=1", 200}, {"before_seq=0&limit=100", 200}, {"before_seq=0&after_seq=0", 400}, {"before_seq=0&before_id=1", 400}, {"after_seq=0&limit=101", 400}, {"after_seq=-1", 400}, {"after_seq=9223372036854775808", 400}, {"after_seq=0&snapshot_seq=2", 409}, {"after_seq=2", 409}, {"before_seq=", 400}, {"after_seq=0&after_seq=1", 400}, {"snapshot_seq=1", 400},
	} {
		response = httptest.NewRecorder()
		r.ServeHTTP(response, httptest.NewRequest("GET", "/api/chatrooms/1/messages?"+tc.query, nil))
		require.Equal(t, tc.status, response.Code, tc.query+": "+response.Body.String())
		if tc.status == 200 {
			require.Contains(t, response.Body.String(), `"high_water":"1"`)
			require.Contains(t, response.Body.String(), `"request_id":"seq-http"`)
		}
	}
}
