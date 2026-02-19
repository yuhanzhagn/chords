package gateway

import (
	"connection/internal/service"
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

type WSConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
	SetWriteDeadline(t time.Time) error
}

type Connection struct {
	Ws   WSConn
	Send chan []byte
}

func ServeWs[T any](
	userID uint32,
	w http.ResponseWriter,
	r *http.Request,
	hub *Hub[T],
	msgService *service.MessageService,
) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	conn := &Connection{
		Ws: ws,
	}

	client := &Client{
		ID:         userID,
		Conn:       conn,
		SendChan:   make(chan []byte, 256),
		wsMsgType:  hub.WSMessageType(),
		msgService: *msgService,
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

		switch {

		case hub.IsJoin(event):
			hub.AddClientToRoom(c.ID, hub.RoomID(event))

		case hub.IsLeave(event):
			hub.RemoveClientFromRoom(c.ID, hub.RoomID(event))

		case hub.IsMessage(event):
			encodedEvent, err := hub.Codec().Encode(event)
			if err != nil {
				log.Println("encode error:", err)
				continue
			}
			c.msgService.HandleIncomingMessage(encodedEvent)

		default:
			// ignore / log
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
