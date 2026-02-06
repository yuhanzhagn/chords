package websocket

import (
	"connection/internal/service"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	//"encoding/json"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins, adjust for production
	},
}

// subscribe, message
type WSMessage struct {
	MsgType string `json:"MsgType"`
	UserID  uint   `json:"UserID"`
	RoomID  uint   `json:"RoomID"`
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

func ServeWs(
	userID uint,
	w http.ResponseWriter,
	r *http.Request,
	hub *Hub,
	msgService service.MessageService,
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
		ID:       userID,
		Conn:     conn,
		SendChan: make(chan []byte, 256),
	}

	// 1. 注册 client（全局唯一真相）
	hub.AddClient(client)
	// 2. 启动 IO goroutine
	go writePump(client)
	go readPump(client, hub)
}

func readPump(c *Client, hub *Hub) {
	defer func() {
		log.Printf("Websocket is closing.")
		c.Conn.Ws.Close()
	}()

	for {
		_, raw, err := c.Conn.Ws.ReadMessage()
		if err != nil {
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.MsgType {

		case "join":
			hub.AddClientToRoom(c.ID, msg.RoomID)

		case "leave":
			hub.RemoveClientFromRoom(c.ID, msg.RoomID)

		case "message":
			// 纯业务消息 → Kafka
			c.msgService.HandleIncomingMessage(raw)

		default:
			// ignore / log
		}
	}

}

func writePump(c *Client) {
	for msg := range c.SendChan {
		c.Conn.Ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := c.Conn.Ws.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("write error:", err)
			break
		}
		log.Printf("Broadcasting to a chatroom")
	}
}
