import React, { useEffect, useRef, useState } from "react";
import ChatSidebar from "./ChatSidebar";
import ChatWindow from "./ChatWindow";
import { useUser } from '../context/UserProvider';
import "./chat.css";
import { v4 as uuidv4 } from "uuid";
import { useNavigate } from 'react-router-dom';
import { RefreshContext } from "./RefreshContext";

const ipaddr = `${process.env.REACT_APP_URL}`
const SOCKET_URL = `ws://${ipaddr}/ws/`; // <-- change to your WebSocket endpoint

const ChatRoom = () => {
  const [chatrooms, setChatrooms] = useState([]);
  const [selectedChatroom, setSelectedChatroom] = useState(null);
  const [messages, setMessages] = useState([]);
  const [chatroomsLoading, setChatroomsLoading] = useState(false);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [error, setError] = useState(null);
  const [messageIndex, setMessageIndex] = useState(0);
  const [showPopup, setShowPopup] = useState(false);
  const [countdown, setCountdown] = useState(3);
  const [refreshKey, setRefreshKey] = useState(0);
  const triggerRefresh = () => setRefreshKey(k => k + 1);
  const socketRef = useRef(null);
  const jwttoken = localStorage.getItem("jwt");
  const navigate = useNavigate();

  const storedUser = localStorage.getItem('user');
  const user = storedUser ? JSON.parse(storedUser) : null;

  const messagesRef = useRef(messages);
  const selectedChatroomRef = useRef(selectedChatroom);

  //const getSelectedChatroom = () => selectedChatroom;

    useEffect(() => {
        messagesRef.current = messages; // keep ref updated
    }, [messages]);
  /* Load chatrooms when component mounts */
  useEffect(() => {
    const loadChatrooms = async () => {
      setChatroomsLoading(true);
      setError(null);

      try {
        const res = await fetch(`http://${ipaddr}/memberships/${user.username}/chatrooms`, {
        method: "GET",
        headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${jwttoken}`, // <-- JWT here
    },
    });
        if (!res.ok) throw new Error(`Failed to fetch chatrooms (status ${res.status})`);

        const data = await res.json();
        setChatrooms(data.data);
        //console.log(data.data);
      } catch (err) {
        setError(err.message);
      } finally {
        setChatroomsLoading(false);
      }
    };

    loadChatrooms();
  }, [refreshKey]);
    

  /* Establish WebSocket connection once */
  useEffect(() => {
    const socket = new WebSocket(SOCKET_URL + user.id);
    socketRef.current = socket;

    socket.onopen = () => {
      console.info("[socket] connection opened");
      socket.send(JSON.stringify({
        MsgType: "AUTH",
        RoomID:0,
        UserID:user.id,
        Message: jwttoken,
        TempID:uuidv4(),
        }));
 
      if (selectedChatroom) {
        socket.send(JSON.stringify({ MsgType: "SUBSCRIBE", RoomID: selectedChatroom.ID, UserID: user.id, Message:"", TempID: uuidv4()}));
      }
        };

    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        console.log("ws on message");
        console.log(payload);
        console.log(selectedChatroomRef.current);
        switch (payload.MsgType) {
          case "MESSAGE_TO_CLIENT":
            if (payload.RoomID === selectedChatroomRef.current?.ID) {
              let modeledMsg = JSON.parse(payload.Message);
              modeledMsg = {
                ...modeledMsg,
                TempID: payload.TempID,
                status: "sent",
                fromself: payload.UserID === user.id        
                }
                console.log("going to replace msg")
              if (payload.UserID !== user.id ){
                    setMessages((prev) => [...prev, modeledMsg]);
                }else{
                    let found = messagesRef.current.find(element=>element?.TempID === payload.TempID);
                    if (found){
                    setMessages(prev =>
                        prev.map(m =>
                        m?.TempID === modeledMsg.TempID
                        ? { ...modeledMsg } // replace pending message with confirmed one
                        : m
                        )
                    );}else{
                        setMessages((prev) => [...prev, modeledMsg]);
                    }
            }
            }
            break;

          // Handle other event types (e.g., system messages, room updates) here.
        case "CLOSING":
            setShowPopup(true);
            break;  
         default:
            break;
        }
      } catch (parseErr) {
        console.error("[socket] failed to parse message", parseErr);
      }
    };

    socket.onerror = (event) => {
      console.error("[socket] error", event);
      setError("WebSocket error! Check console for details.");
    };

    socket.onclose = () => {
    if (selectedChatroom) {
        socket.send(JSON.stringify({ MsgType: "UNSUBSCRIBE", RoomID: selectedChatroom.ID, UserID: user.id, Message:"", TempID: uuidv4() }));
      }
      console.info("[socket] connection closed");
        //navigate("/login");
    };

    return () => {
      socket.close();
      socketRef.current = null;
    };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

//handle popup window on websocket close
    useEffect(() => {
    if (!showPopup) return;

    const interval = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) {
          clearInterval(interval);
          navigate("/login"); // redirect URL
        }
        return c - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [showPopup]);



  /* When chatroom changes, fetch history and notify socket */
  const handleSelectChatroom = async (room) => {

      if (selectedChatroom) {
        socketRef.current.send(JSON.stringify({ MsgType: "UNSUBSCRIBE", RoomID: selectedChatroom.ID, UserID: user.id, Message:"", TempID: uuidv4() }));
      }

    setSelectedChatroom(room);
    selectedChatroomRef.current = room;
    setMessages([]);
    setMessagesLoading(true);
    setError(null);

    try {
    const res = await fetch(`http://${ipaddr}/chatrooms/${room.ID}/messages`, {
    method: "GET", // default is GET, can omit
     headers: {
        "Content-Type": "application/json",
        "Authorization": `Bearer ${jwttoken}`, // Add JWT here
    },
    });
      if (!res.ok) throw new Error(`Failed to fetch messages (status ${res.status})`);

      let data = await res.json();
      data = data.map(m => ({
        ...m,
        status: "sent", // keep existing if any, else default to "sent"
        fromself: m.UserID === user.id,
        TempID: uuidv4(),
      }));
      setMessages(data);

      if (socketRef.current?.readyState === WebSocket.OPEN) {
        socketRef.current.send(JSON.stringify({ MsgType: "SUBSCRIBE", RoomID: room.ID, UserID: user.id, Message:"", TempID: uuidv4() }));
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setMessagesLoading(false);
    }
  };

  /* Send message over socket */
  const handleSendMessage = (text) => {
    if (!selectedChatroom) return;
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      setError("Socket not connected; cannot send message.");
      return;
    }

    let payload = {
      MsgType: "MESSAGE_TO_SERVER",
      RoomID: selectedChatroom.ID,
      UserID: user.id,
      Message: text,
      TempID: uuidv4(), 
    };
    // waiting for the check from the server
    // setPendingMsg([...pendingMsg, payload]);
    socketRef.current.send(JSON.stringify(payload));
    let modeledMsg = {
        ID:-1,
        UserID: user.id,
        ChatRoomID: selectedChatroom.ID,
        Content: payload.Message,
        CreatedAt: new Date().toISOString(),
        TempID: payload.TempID, 
        status: "pending",
        fromself: true,
    };
    setMessages(prev => [...prev, modeledMsg]);
  };

  return (
    <div className="chat-layout">
         <RefreshContext.Provider value={triggerRefresh}>
      <ChatSidebar
        chatrooms={chatrooms}
        loading={chatroomsLoading}
        selectedChatroom={selectedChatroom}
        onSelectChatroom={handleSelectChatroom}
      />
         </RefreshContext.Provider>


      <ChatWindow
        chatroom={selectedChatroom}
        messages={messages}
        loading={messagesLoading}
        onSendMessage={handleSendMessage}
      />

      {showPopup && (
        <div className="popup-content">
          <div className="popup-window">
            <h2>Connection lost</h2>
            <p>You will be redirected in {countdown} seconds…</p>
          </div>
        </div>
      )}

     {/* {error && <div className="chat-error-banner">{error}</div>}*/ }
    </div>
  );
};

export default ChatRoom;
