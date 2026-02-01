package websocket

import (
    "github.com/gorilla/websocket"
    "log"
    "net/http"
    "time"
	//"errors"

	"backend/internal/service"
	"backend/internal/model"
	"backend/utils"
	"encoding/json"
	
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // allow all origins, adjust for production
    },
}

//subscribe, message
type WSMessage struct {
    MsgType  string `json:"MsgType"`
    UserID   uint    `json:"UserID"`
    RoomID   uint    `json:"RoomID"`
    Message  string `json:"Message"`
	TempID	 string `json:"TempID"`
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

func ServeWs(userID uint, w http.ResponseWriter, r *http.Request, msgService service.MessageService) {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("upgrade error:", err)
        return
    }

    conn := &Connection{
        Ws:   ws,
        Send: make(chan []byte, 256),
    }

    client := &Client{
        ID:       userID,
        Conn:     conn,
        SendChan: conn.Send,
		msgService: msgService,
    }

	setClient(client)
	log.Printf("ws serving")
    //GlobalHub.AddClient(roomID, client)
    go writePump(client)
    go readPump(client)
}

func connectChatroom(id uint, roomID uint) {
	client := getClient(id)
	if client == nil {
		log.Printf("Client %d not found\n", id)
		return
	}
	GlobalHub.AddClient(roomID, client)	
	log.Printf("Client %d connected to room %d\n", id, roomID)
}


func readAndParseWSMessage(c *Client) *WSMessage{
	_, msgBytes, err := c.Conn.Ws.ReadMessage()
    if err != nil {
		log.Println("websocket read error:", err)
		return nil
    }

    var wsMsg WSMessage
    if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
        log.Println("json parsing error:", err)
		return nil
    }
	return &wsMsg
}	
 

func readPump(c *Client) {
    defer func() {
		// how to unsubsribe
        //GlobalHub.RemoveClient(c.RoomID, c)
		log.Printf("Websocket is closing.")
		removeClient(c.ID)
        c.Conn.Ws.Close()
    }()
	isauth := false
	
 	for !isauth {
		wsMsg := readAndParseWSMessage(c)
        if wsMsg == nil{
            break
        }
        // parse and validate auth
		if wsMsg.MsgType == "AUTH"{
            _, err := utils.ParseJWT(wsMsg.Message)
            if err != nil {
                // invalid token → reject connection
                log.Println("Unauthorized:", err)
                c.Conn.Ws.Close();
            }
			isauth = true
    	}
	}

    for {
		wsMsg := readAndParseWSMessage(c)
		if wsMsg == nil{
			break	
		}
        /*
		_, msgBytes, err := c.Conn.Ws.ReadMessage()
        if err != nil {
            log.Println("read error:", err)
            break
        }
		
		var wsMsg WSMessage
    	if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
			log.Println("json parsing error:", err)
   		}
		*/

		//identify message type
		if wsMsg.MsgType == "SUBSCRIBE" {
			GlobalHub.AddClient(wsMsg.RoomID, getClient(wsMsg.UserID))		
		} else if wsMsg.MsgType == "MESSAGE_TO_SERVER"{
			msg := &model.Message{
            	UserID: wsMsg.UserID,
            	ChatRoomID: wsMsg.RoomID,
            	Content: string(wsMsg.Message),
        	}
        	err := c.msgService.CreateMessage(msg)
        	if err != nil {
            	log.Println("write into db error:", err)
            	break
        	}
			log.Println("write message:", msg.Content)
		// modify the boardcast function
			// MESSAGE field of WSMessage
			tempBytes, err := json.Marshal(msg)
			if err != nil {
   				log.Println("failed to marshal msg:", err)
    			break
			}
			sendMsg := &WSMessage{
				MsgType: "MESSAGE_TO_CLIENT",
				RoomID: wsMsg.RoomID,
				UserID: wsMsg.UserID,
				Message: string(tempBytes),
				TempID: wsMsg.TempID, 
			} 
			sendMsgBytes, err := json.Marshal(sendMsg)	
			if err != nil {
				log.Println("json marshal error:", err)
			}
			// Broadcast message to everyone in room
       		GlobalHub.Broadcast(wsMsg.RoomID, sendMsgBytes)
		}else if wsMsg.MsgType == "UNSUBSCRIBE"{
			GlobalHub.RemoveClient(wsMsg.RoomID, c)	
		}
		/*
			else if wsMsg.MsgType == "AUTH"{
			_, err := utils.ParseJWT(wsMsg.Message)
			if err != nil {
    			// invalid token → reject connection
    			log.Println("Unauthorized:", err)
				c.Conn.Ws.Close();				
			}*/
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


func Disconnect(id uint) {
	go func(){
	c := getClient(id)
	if c == nil {
		log.Println("trying to get the client by userID by not fount")
		return
	}
	sendMsg := &WSMessage{
    	MsgType: "CLOSING",
    	RoomID: 0,
        UserID: id,
        Message: "",
      	TempID: "",
    }
	sendMsgBytes, err := json.Marshal(sendMsg)
    if err != nil {
     	log.Println("json marshal error:", err)
		return
   	}

	select {
		case c.SendChan <- sendMsgBytes:
		default:
    		log.Println("send channel full, dropping CLOSING message")
	}
	time.Sleep(3 * time.Second)
	removeClient(c.ID)
    c.Conn.Ws.Close()
	}()
}
