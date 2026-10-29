package fanout

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fanout/internal/fanoutauth"
	kafkapb "fanout/proto/kafka"
)

const testToken = "isolated-fanout-test-credential-32-bytes"

type fakeRegistry struct {
	addr string
	err  error
}

func (r fakeRegistry) RoomUsers(context.Context, uint32) ([]uint32, error) {
	return []uint32{42}, r.err
}
func (r fakeRegistry) UserGateways(context.Context, []uint32) (map[uint32]string, error) {
	return map[uint32]string{42: r.addr}, r.err
}

func TestDispatcherSendsDedicatedCredentialAndPayload(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/fanout" || r.Header.Get(fanoutauth.Header) != testToken || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid service request")
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("user credential forwarded")
		}
		var req FanoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.RoomID != 7 || len(req.UserIDs) != 0 || req.Event == nil || string(req.Event.Content) != "private" {
			t.Error("payload changed")
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	d, err := NewDispatcher(fakeRegistry{addr: server.URL}, server.Client(), Config{Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Dispatch(context.Background(), &kafkapb.KafkaEvent{RoomId: 7, Content: []byte("private")}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("expected one authenticated POST")
	}
}

func TestDispatcherRejectsHTTPFailuresWithoutLeakingResponse(t *testing.T) {
	for _, status := range []int{400, 401, 403, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); _, _ = w.Write([]byte(testToken)) }))
			defer server.Close()
			d, err := NewDispatcher(fakeRegistry{addr: server.URL}, server.Client(), Config{Token: testToken})
			if err != nil {
				t.Fatal(err)
			}
			err = d.Dispatch(context.Background(), &kafkapb.KafkaEvent{RoomId: 7})
			if err == nil || strings.Contains(err.Error(), testToken) {
				t.Fatalf("HTTP failure lost or secret leaked: %v", err)
			}
		})
	}
}

func TestDispatcherNeverFollowsRedirects(t *testing.T) {
	var forwarded atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded.Add(1); w.WriteHeader(204) }))
	defer target.Close()
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, status) }))
			defer origin.Close()
			client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { t.Error("caller redirect policy used"); return nil }}
			d, err := NewDispatcher(fakeRegistry{addr: origin.URL}, client, Config{Token: testToken})
			if err != nil {
				t.Fatal(err)
			}
			if err = d.Dispatch(context.Background(), &kafkapb.KafkaEvent{RoomId: 7}); err == nil {
				t.Fatal("redirect treated as success")
			}
			if client.CheckRedirect == nil {
				t.Fatal("caller client mutated")
			}
		})
	}
	if forwarded.Load() != 0 {
		t.Fatal("credential-bearing redirect reached another target")
	}
}

func TestDispatcherRejectsBadConstructionAndRegistryFailure(t *testing.T) {
	for _, token := range []string{"", "short", strings.Repeat("x", 32) + "\n"} {
		if _, err := NewDispatcher(nil, http.DefaultClient, Config{Token: token}); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	if _, err := NewDispatcher(nil, nil, Config{Token: testToken}); err == nil {
		t.Fatal("nil client accepted")
	}
	expected := errors.New("registry unavailable")
	d, err := NewDispatcher(fakeRegistry{err: expected}, http.DefaultClient, Config{Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if err = d.Dispatch(context.Background(), &kafkapb.KafkaEvent{RoomId: 7}); !errors.Is(err, expected) {
		t.Fatal(err)
	}
}

type manyUserRegistry struct {
	users    []uint32
	gateways map[uint32]string
}

func (r manyUserRegistry) RoomUsers(context.Context, uint32) ([]uint32, error) { return r.users, nil }
func (r manyUserRegistry) UserGateways(context.Context, []uint32) (map[uint32]string, error) {
	return r.gateways, nil
}

func TestRoomBroadcastOneBoundedRequestPerSelectedGateway(t *testing.T) {
	var calls [2]atomic.Int32
	servers := make([]*httptest.Server, 2)
	for i := range servers {
		index := i
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls[index].Add(1)
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if len(raw) > kafkapb.MaxTransportBytes {
				t.Error("unbounded envelope")
			}
			var req FanoutRequest
			if err := json.Unmarshal(raw, &req); err != nil {
				t.Error(err)
			}
			if len(req.UserIDs) != 0 || bytes.Contains(raw, []byte("user_ids")) || req.RoomID != 7 {
				t.Error("room broadcast payload must omit recipients")
			}
			if req.Event == nil || len(req.Event.Content) != kafkapb.MaxContentBytes {
				t.Error("event changed")
			}
			w.WriteHeader(204)
		}))
		defer servers[i].Close()
	}
	users := make([]uint32, 100001)
	gateways := make(map[uint32]string, len(users))
	for i := range users {
		users[i] = uint32(4000000000 + i)
		if i < len(users)-1 {
			gateways[users[i]] = servers[i%2].URL
		}
	}
	d, err := NewDispatcher(manyUserRegistry{users, gateways}, http.DefaultClient, Config{Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), &kafkapb.KafkaEvent{RoomId: 7, Content: bytes.Repeat([]byte("x"), kafkapb.MaxContentBytes)}); err != nil {
		t.Fatal(err)
	}
	for i := range calls {
		if calls[i].Load() != 1 {
			t.Fatal("each selected gateway must receive exactly one room broadcast")
		}
	}
}

func TestFallbackRetainsRecipientsAndFinalBodyIsBounded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req FanoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.RoomID != 0 || len(req.UserIDs) != 1 || req.UserIDs[0] != 42 {
			t.Error("fallback changed")
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	d, err := NewDispatcher(fakeRegistry{addr: server.URL}, server.Client(), Config{Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), &kafkapb.KafkaEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := d.Dispatch(context.Background(), &kafkapb.KafkaEvent{RoomId: 7, Content: bytes.Repeat([]byte("x"), kafkapb.MaxTransportBytes)}); !errors.Is(err, kafkapb.ErrMessageTooLarge) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("oversize envelope sent")
	}
}
