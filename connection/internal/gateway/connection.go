package gateway

import (
	"connection/internal/handler"
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins, adjust for production
	},
}

var (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = int64(1 << 20)
)

// subscribe, message
type WSMessage struct {
	MsgType string `json:"MsgType"`
	UserID  uint32 `json:"UserID"`
	RoomID  uint32 `json:"RoomID"`
	Message string `json:"Message"`
	TempID  string `json:"TempID"`
}

// InboundEvent wraps a decoded event with transport metadata.
type InboundEvent[T any] struct {
	ClientID uint32
	Event    T
}

type WSConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
	SetWriteDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetReadLimit(limit int64)
	SetPongHandler(h func(string) error)
}

type Connection struct {
	Ws             WSConn
	Send           chan []byte
	inboundHandler handler.HandlerFunc
	Request        *http.Request
	Ctx            context.Context
	Cancel         context.CancelFunc
}

func NewConnection(ws WSConn, inboundHandler handler.HandlerFunc, request *http.Request) (*Connection, error) {
	if ws == nil {
		return nil, errors.New("ws connection is required")
	}
	if inboundHandler == nil {
		return nil, errors.New("inbound handler is required")
	}
	if request == nil {
		return nil, errors.New("http request is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Ws:             ws,
		inboundHandler: inboundHandler,
		Request:        request,
		Ctx:            ctx,
		Cancel:         cancel,
	}, nil
}

func ServeWs[T any](
	userID uint32,
	w http.ResponseWriter,
	r *http.Request,
	hub *Hub[T],
	inboundHandler handler.HandlerFunc,
) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	conn, err := NewConnection(ws, inboundHandler, r)
	if err != nil {
		log.Println("connection setup error:", err)
		_ = ws.Close()
		return
	}

	client := &Client{
		ID:        userID,
		Conn:      conn,
		SendChan:  make(chan []byte, 256),
		wsMsgType: hub.WSMessageType(),
	}

	// 1. 注册 client（全局唯一真相）
	hub.AddClient(client)
	// 2. 启动 IO goroutine
	go writePump(client)
	go readPump(client, hub)
}

func readPump[T any](c *Client, hub *Hub[T]) {
	defer func() {
		log.Printf("Websocket is closing.")
		if c.Conn.Cancel != nil {
			c.Conn.Cancel()
		}
		if hub != nil {
			userID, groupIDs := hub.RemoveClientAndGroups(c.ID)
			hub.handleDisconnect(c.ID, userID, groupIDs)
		}
		c.Close()
	}()

	c.Conn.Ws.SetReadLimit(maxMessageSize)
	if err := c.Conn.Ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("set read deadline error: %v", err)
	}
	c.Conn.Ws.SetPongHandler(func(string) error {
		return c.Conn.Ws.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, raw, err := c.Conn.Ws.ReadMessage()
		if err != nil {
			return
		}
		log.Printf("%x\n", raw)

		event, err := hub.Codec().Decode(raw)
		if err != nil {
			log.Println("decode error:", err)
			continue
		}

		err = c.Conn.inboundHandler(&handler.Context{
			Context:    c.Conn.Ctx,
			ClientID:   c.ID,
			Event:      InboundEvent[T]{ClientID: c.ID, Event: event},
			ReceivedAt: time.Now(),
			Values: map[string]any{
				handler.RequestContextKey: c.Conn.Request,
			},
		})
		if err != nil {
			log.Println("inbound handler error:", err)
		}
	}

}

func writePump(c *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.SendChan:
			if !ok {
				return
			}
			if err := c.Conn.Ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("set write deadline error: %v", err)
			}
			if err := c.Conn.Ws.WriteMessage(c.wsMsgType, msg); err != nil {
				log.Println("write error:", err)
				return
			}
			log.Printf("Broadcasting to a chatroom")
		case <-ticker.C:
			if err := c.Conn.Ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("set write deadline error: %v", err)
			}
			if err := c.Conn.Ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("ping error:", err)
				return
			}
		}
	}
}
