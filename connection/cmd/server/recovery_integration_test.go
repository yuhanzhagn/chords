package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connection/internal/source"
	kafkapb "connection/proto/kafka"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronizedBuffer) Write(raw []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(raw)
}
func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func startRecoveryFixture(t *testing.T, module string, env []string) map[string]string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "../../..", module)
	binary := filepath.Join(t.TempDir(), "fixture")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./test/recoveryfixture")
	build.Dir = dir
	raw, err := build.CombinedOutput()
	require.NoError(t, err, "%s", raw)
	command := exec.Command(binary)
	command.Env = append(os.Environ(), env...)
	stdin, err := command.StdinPipe()
	require.NoError(t, err)
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	var stderr synchronizedBuffer
	command.Stderr = &stderr
	require.NoError(t, command.Start())
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	t.Cleanup(func() {
		_ = stdin.Close()
		select {
		case err := <-done:
			require.NoError(t, err, "%s", stderr.String())
		case <-time.After(5 * time.Second):
			_ = command.Process.Kill()
			<-done
			t.Error("fixture shutdown timeout")
		}
	})
	reader := bufio.NewReader(stdout)
	line, err := reader.ReadBytes('\n')
	require.NoError(t, err, "%s", stderr.String())
	var metadata map[string]string
	require.NoError(t, json.Unmarshal(line, &metadata), "%s", line)
	go io.Copy(io.Discard, reader)
	return metadata
}

type faultTransport struct {
	base http.RoundTripper
	mode atomic.Int32
}

func (f *faultTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Path != "/api/messages" {
		return f.base.RoundTrip(r)
	}
	mode := f.mode.Swap(0)
	if mode == 2 {
		return nil, context.DeadlineExceeded
	} // before any backend transaction
	if mode >= 400 {
		return &http.Response{StatusCode: int(mode), Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	}
	response, err := f.base.RoundTrip(r)
	if mode == 1 && err == nil {
		_ = response.Body.Close()
		return nil, errors.New("test: response lost after commit")
	}

	return response, err
}

func TestRecoveryHTTPGatewayRelayFanout(t *testing.T) {
	if os.Getenv("CHORDS_RUN_RECOVERY_PROBE") != "1" {
		t.Skip("explicit local multi-module process probe; see docs/testing.md")
	}
	const serviceToken = "isolated-s2-fanout-test-token-32bytes"
	hub := newHub(newEventCodec("protobuf"))
	assign := groupAssignmentHandler(hub, nil, "")
	authorizer := newBackendAuthorizer("")
	faults := &faultTransport{base: http.DefaultTransport}
	authorizer.client.Transport = faults
	fanoutSource, err := source.NewFanoutHTTPHandler(hub, "", serviceToken)
	require.NoError(t, err)
	mux := http.NewServeMux()
	mux.Handle("/ws", WsHandler(hub, assign, authorizer.authenticator(assign)))
	mux.Handle("/fanout", fanoutSource)
	gatewayServer := httptest.NewServer(mux)
	defer gatewayServer.Close()
	worker := startRecoveryFixture(t, "fanout", []string{"CHORDS_PROBE_GATEWAY=" + gatewayServer.URL, "FANOUT_TOKEN=" + serviceToken})
	var brokerDown, dropNotification atomic.Bool
	brokerDown.Store(true)
	published := make(chan *kafkapb.KafkaEvent, 256)
	broker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if brokerDown.Load() {
			w.WriteHeader(503)
			return
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		var e kafkapb.KafkaEvent
		if proto.Unmarshal(raw, &e) != nil {
			w.WriteHeader(400)
			return
		}
		published <- &e
		if dropNotification.Load() {
			w.WriteHeader(204)
			return
		}
		response, err := http.Post(worker["url"], "application/octet-stream", bytes.NewReader(raw))
		if err != nil {
			w.WriteHeader(503)
			return
		}
		defer response.Body.Close()
		w.WriteHeader(response.StatusCode)
	}))
	defer broker.Close()
	backend := startRecoveryFixture(t, "backend", []string{"CHORDS_PROBE_BROKER=" + broker.URL})
	authorizer.url = backend["url"]
	dial := func() *websocket.Conn {
		ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(gatewayServer.URL, "http")+"/ws", nil)
		require.NoError(t, err)
		t.Cleanup(func() { ws.Close() })
		require.NoError(t, ws.WriteJSON(map[string]string{"type": "auth", "token": backend["token"]}))
		var ready map[string]string
		require.NoError(t, ws.ReadJSON(&ready))
		require.Equal(t, "ready", ready["type"])
		return ws
	}
	ws := dial()
	send := func(e *kafkapb.KafkaEvent) {
		raw, err := proto.Marshal(e)
		require.NoError(t, err)
		require.NoError(t, ws.WriteMessage(websocket.BinaryMessage, raw))
	}
	read := func() *kafkapb.KafkaEvent {
		require.NoError(t, ws.SetReadDeadline(time.Now().Add(5*time.Second)))
		_, raw, err := ws.ReadMessage()
		require.NoError(t, err)
		var e kafkapb.KafkaEvent
		require.NoError(t, proto.Unmarshal(raw, &e))
		return &e
	}
	send(&kafkapb.KafkaEvent{MsgType: "join", RoomId: 1})
	input := &kafkapb.KafkaEvent{MsgType: "message", RoomId: 1, RequestId: "commit-response-lost", TempId: "temp", Content: []byte("hello")}
	faults.mode.Store(1)
	send(input)
	unknown := read()
	require.Equal(t, "send_error", unknown.MsgType)
	require.Contains(t, string(unknown.Content), "unknown")
	send(input)
	ack := read()
	require.Equal(t, "persisted", ack.MsgType)
	require.EqualValues(t, 1, ack.RoomSeq)
	// Broker remains down: ACK was based solely on SQLite COMMIT.
	brokerDown.Store(false)
	notification := read()
	require.Equal(t, "message", notification.MsgType)
	require.Equal(t, ack.EventId, notification.EventId)
	require.Equal(t, ack.ServerMessageId, notification.ServerMessageId)
	firstPublished := <-published
	require.Equal(t, ack.EventId, firstPublished.EventId)
	faults.mode.Store(2)
	input.RequestId = "before-transaction-timeout"
	send(input)
	require.Equal(t, "send_error", read().MsgType)
	send(input)
	var second *kafkapb.KafkaEvent
	for i := 0; i < 2; i++ {
		event := read()
		if event.MsgType == "persisted" {
			second = event
			break
		}
	}
	require.NotNil(t, second)
	require.EqualValues(t, 2, second.RoomSeq)
	// Consume that request's notification before exercising rejection frames.
	if len(published) == 0 {
		select {
		case <-published:
		case <-time.After(3 * time.Second):
			t.Fatal("second relay timeout")
		}
	}
	// Reconnect avoids relying on ACK/notification ordering in the old socket queue.
	ws.Close()
	ws = dial()
	send(&kafkapb.KafkaEvent{MsgType: "join", RoomId: 1})
	dropNotification.Store(true)
	for _, code := range []int32{409, 413, 503} {
		faults.mode.Store(code)
		input.RequestId = fmt.Sprint("error-", code)
		send(input)
		event := read()
		require.Equal(t, "send_error", event.MsgType)
		require.Contains(t, string(event.Content), fmt.Sprint(code))
	}
	// Explicit backend failures never become a persistence ACK.
	dropNotification.Store(true)
	post := func(path string, body any) *http.Response {
		raw, _ := json.Marshal(body)
		request, err := http.NewRequest(http.MethodPost, backend["url"]+path, bytes.NewReader(raw))
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+backend["token"])
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		return response
	}
	for i := 0; i < 65; i++ {
		response := post("/api/messages", map[string]any{"user_id": 1, "chat_room_id": 1, "content": "offline", "request_id": fmt.Sprint("offline-", i)})
		require.Equal(t, 201, response.StatusCode)
		response.Body.Close()
	}
	cursor := "0"
	snapshot := ""
	total := 0
	pages := 0
	for {
		endpoint := backend["url"] + "/api/chatrooms/1/messages?after_seq=" + cursor + "&limit=30"
		if snapshot != "" {
			endpoint += "&snapshot_seq=" + snapshot
		}
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+backend["token"])
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		require.Equal(t, 200, response.StatusCode)
		var page struct {
			Messages []json.RawMessage `json:"messages"`
			High     string            `json:"high_water"`
			Next     string            `json:"next_cursor"`
			More     bool              `json:"has_more"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
		response.Body.Close()
		snapshot = page.High
		cursor = page.Next
		total += len(page.Messages)
		pages++
		if !page.More {
			break
		}
	}
	require.GreaterOrEqual(t, total, 67)
	require.GreaterOrEqual(t, pages, 3)
	response := post("/api/probe/remove-member", nil)
	require.Equal(t, 204, response.StatusCode)
	response.Body.Close()
	input.RequestId = "non-member"
	send(input)
	for {
		event := read()
		if event.MsgType == "send_error" {
			require.Contains(t, string(event.Content), "403")
			break
		}
	}
	response = post("/api/probe/revoke", nil)
	require.Equal(t, 204, response.StatusCode)
	response.Body.Close()
	input.RequestId = "revoked"
	send(input)
	event := read()
	require.Equal(t, "send_error", event.MsgType)
	require.Contains(t, string(event.Content), "401")
}
