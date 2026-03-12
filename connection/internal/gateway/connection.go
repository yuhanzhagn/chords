package gateway

import (
	"context"
	"connection/internal/handler"
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
		c.Conn.Ws.Close()
	}()

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
	for msg := range c.SendChan {
		c.Conn.Ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := c.Conn.Ws.WriteMessage(c.wsMsgType, msg)
		if err != nil {
			log.Println("write error:", err)
			break
		}
		log.Printf("Broadcasting to a chatroom")
	}
}
