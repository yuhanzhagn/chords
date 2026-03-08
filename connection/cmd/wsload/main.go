package main

import (
	kafkapb "connection/proto/kafka"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type simClient struct {
	id   uint32
	conn *websocket.Conn
}

type stats struct {
	connected   atomic.Int64
	connectFail atomic.Int64
	sent        atomic.Int64
	sendFail    atomic.Int64
	received    atomic.Int64
	recvDecode  atomic.Int64
}

var eventSeq atomic.Uint64

func main() {
	var (
		wsURL        string
		users        int
		roomID       uint
		messagesEach int
		sendJitter   time.Duration
		joinWait     time.Duration
		settleWait   time.Duration
	)

	flag.StringVar(&wsURL, "url", "ws://localhost:8000/ws", "WebSocket endpoint")
	flag.IntVar(&users, "users", 200, "number of concurrent users")
	flag.UintVar(&roomID, "room", 1, "chat room id")
	flag.IntVar(&messagesEach, "messages", 3, "messages sent per user")
	flag.DurationVar(&sendJitter, "send-jitter", 100*time.Millisecond, "max random delay before each send")
	flag.DurationVar(&joinWait, "join-wait", 800*time.Millisecond, "wait after all joins before sending messages")
	flag.DurationVar(&settleWait, "settle-wait", 3*time.Second, "wait after sends to receive broadcasts")
	flag.Parse()

	if users <= 0 {
		log.Fatal("users must be > 0")
	}
	if messagesEach < 0 {
		log.Fatal("messages must be >= 0")
	}
	if roomID == 0 {
		log.Fatal("room must be > 0")
	}

	log.Printf("wsload starting: users=%d room=%d messages/user=%d url=%s", users, roomID, messagesEach, wsURL)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	start := time.Now()
	st := &stats{}

	clients := connectClients(ctx, wsURL, users, st)
	if len(clients) == 0 {
		log.Fatal("no clients connected")
	}
	log.Printf("connected=%d failed=%d", st.connected.Load(), st.connectFail.Load())

	var readWG sync.WaitGroup
	for i := range clients {
		readWG.Add(1)
		go readLoop(ctx, clients[i], st, &readWG)
	}

	for i := range clients {
		if err := sendEvent(clients[i].conn, newEvent(clients[i].id, uint32(roomID), "join", nil)); err != nil {
			st.sendFail.Add(1)
			log.Printf("join send failed user=%d err=%v", clients[i].id, err)
			continue
		}
		st.sent.Add(1)
	}

	time.Sleep(joinWait)

	for msgIdx := 0; msgIdx < messagesEach; msgIdx++ {
		for i := range clients {
			if sendJitter > 0 {
				time.Sleep(time.Duration(rand.Int63n(sendJitter.Nanoseconds() + 1)))
			}
			payload := []byte(fmt.Sprintf("u%d message %d", clients[i].id, msgIdx+1))
			if err := sendEvent(clients[i].conn, newEvent(clients[i].id, uint32(roomID), "message", payload)); err != nil {
				st.sendFail.Add(1)
				continue
			}
			st.sent.Add(1)
		}
	}

	time.Sleep(settleWait)

	for i := range clients {
		if err := sendEvent(clients[i].conn, newEvent(clients[i].id, uint32(roomID), "leave", nil)); err != nil {
			st.sendFail.Add(1)
		} else {
			st.sent.Add(1)
		}
		_ = clients[i].conn.Close()
	}

	cancel()
	readWG.Wait()

	elapsed := time.Since(start)
	totalMessages := int64(len(clients) * messagesEach)
	log.Printf("done in %s", elapsed.Truncate(time.Millisecond))
	log.Printf("connected=%d connect_fail=%d", st.connected.Load(), st.connectFail.Load())
	log.Printf("events_sent=%d send_fail=%d chat_messages_target=%d", st.sent.Load(), st.sendFail.Load(), totalMessages)
	log.Printf("events_received=%d recv_decode_fail=%d", st.received.Load(), st.recvDecode.Load())
}

func connectClients(ctx context.Context, wsURL string, users int, st *stats) []simClient {
	clients := make([]simClient, 0, users)
	var mu sync.Mutex
	var wg sync.WaitGroup

	dialer := websocket.Dialer{HandshakeTimeout: 8 * time.Second}

	for i := 1; i <= users; i++ {
		wg.Add(1)
		userID := uint32(i)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}

			header := http.Header{}
			conn, _, err := dialer.DialContext(ctx, wsURL, header)
			if err != nil {
				st.connectFail.Add(1)
				return
			}
			st.connected.Add(1)

			mu.Lock()
			clients = append(clients, simClient{id: userID, conn: conn})
			mu.Unlock()
		}()
	}

	wg.Wait()
	return clients
}

func readLoop(ctx context.Context, c simClient, st *stats, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		ev := &kafkapb.KafkaEvent{}
		if err := proto.Unmarshal(raw, ev); err != nil {
			st.recvDecode.Add(1)
			continue
		}
		st.received.Add(1)
	}
}

func newEvent(userID, roomID uint32, msgType string, content []byte) *kafkapb.KafkaEvent {
	now := time.Now()
	seq := eventSeq.Add(1)
	id := uint64(now.UnixNano()) ^ (uint64(userID) << 16) ^ seq

	return &kafkapb.KafkaEvent{
		Id:        id,
		UserId:    userID,
		RoomId:    roomID,
		MsgType:   msgType,
		Content:   content,
		TempId:    fmt.Sprintf("%d", id),
		CreatedAt: now.UnixMilli(),
	}
}

func sendEvent(conn *websocket.Conn, event *kafkapb.KafkaEvent) error {
	raw, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, raw)
}
