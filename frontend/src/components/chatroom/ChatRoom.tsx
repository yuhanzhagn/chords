import { useEffect, useMemo, useRef, useState } from 'react';
import ChatSidebar from './ChatSidebar';
import ChatWindow from './ChatWindow';
import './chat.css';
import { v4 as uuidv4 } from 'uuid';
import { useNavigate } from 'react-router-dom';
import { RefreshContext } from './RefreshContext';
import type { ChatMessage } from './MessageList';

interface ChatRoomProps {
  roomId?: string;
  userId?: string;
}

interface Chatroom {
  ID: number;
  Name: string;
}

interface UserInfo {
  id: string | number;
  username: string;
}

interface ChatroomResponse {
  data: Chatroom[];
}

interface SocketPayload {
  MsgType: string;
  RoomID: number;
  UserID: string | number;
  Message?: string;
  Content?: string;
  TempID: string;
  [key: string]: unknown;
}

const ipaddr = `${process.env.REACT_APP_URL}`;
const SOCKET_URL = 'ws://localhost:8000/ws';

const ChatRoom = (_props: ChatRoomProps) => {
  const [chatrooms, setChatrooms] = useState<Chatroom[]>([]);
  const [selectedChatroom, setSelectedChatroom] = useState<Chatroom | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [chatroomsLoading, setChatroomsLoading] = useState(false);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showPopup, setShowPopup] = useState(false);
  const [countdown, setCountdown] = useState(3);
  const [refreshKey, setRefreshKey] = useState(0);
  const triggerRefresh = () => setRefreshKey((k) => k + 1);

  const socketRef = useRef<WebSocket | null>(null);
  const messagesRef = useRef<ChatMessage[]>([]);
  const selectedChatroomRef = useRef<Chatroom | null>(null);

  const jwttoken = localStorage.getItem('jwt');
  const navigate = useNavigate();

  const storedUser = localStorage.getItem('user');
  const user = useMemo<UserInfo | null>(() => (storedUser ? JSON.parse(storedUser) : null), [storedUser]);

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  useEffect(() => {
    const loadChatrooms = async () => {
      if (!user?.username || !jwttoken) {
        return;
      }

      setChatroomsLoading(true);
      setError(null);

      try {
        const res = await fetch(`http://${ipaddr}/memberships/${user.username}/chatrooms`, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${jwttoken}`,
          },
        });
        if (!res.ok) throw new Error(`Failed to fetch chatrooms (status ${res.status})`);

        const data: ChatroomResponse = await res.json();
        setChatrooms(data.data || []);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : 'Failed to fetch chatrooms');
      } finally {
        setChatroomsLoading(false);
      }
    };

    loadChatrooms();
  }, [refreshKey, jwttoken, user?.username]);

  useEffect(() => {
    if (!user || !jwttoken) {
      return;
    }

    const socket = new WebSocket(SOCKET_URL);
    socketRef.current = socket;

    socket.onopen = () => {
      console.info('[socket] connection opened');
      const authPayload: SocketPayload = {
        MsgType: 'AUTH',
        RoomID: 0,
        UserID: user.id,
        Message: jwttoken,
        TempID: uuidv4(),
      };
      socket.send(JSON.stringify(authPayload));

      if (selectedChatroomRef.current) {
        const joinPayload: SocketPayload = {
          MsgType: 'join',
          RoomID: selectedChatroomRef.current.ID,
          UserID: user.id,
          Message: '',
          TempID: uuidv4(),
        };
        socket.send(JSON.stringify(joinPayload));
      }
    };

    socket.onmessage = (event: MessageEvent<string>) => {
      try {
        const payload = JSON.parse(event.data) as SocketPayload;

        switch (payload.MsgType) {
          case 'message':
            if (payload.RoomID === selectedChatroomRef.current?.ID) {
              const modeledMsg: ChatMessage = {
                ...(payload as unknown as ChatMessage),
                TempID: payload.TempID,
                status: 'sent',
                fromself: payload.UserID === user.id,
              };

              if (payload.UserID !== user.id) {
                setMessages((prev) => [...prev, modeledMsg]);
              } else {
                const found = messagesRef.current.find((element) => element?.TempID === payload.TempID);
                if (found) {
                  setMessages((prev) =>
                    prev.map((m) => (m?.TempID === modeledMsg.TempID ? { ...modeledMsg } : m)),
                  );
                } else {
                  setMessages((prev) => [...prev, modeledMsg]);
                }
              }
            }
            break;
          case 'close':
            setShowPopup(true);
            break;
          default:
            break;
        }
      } catch (parseErr: unknown) {
        console.error('[socket] failed to parse message', parseErr);
      }
    };

    socket.onerror = (event: Event) => {
      console.error('[socket] error', event);
      setError('WebSocket error! Check console for details.');
    };

    socket.onclose = () => {
      console.info('[socket] connection closed');
    };

    return () => {
      socket.close();
      socketRef.current = null;
    };
  }, [jwttoken, user]);

  useEffect(() => {
    if (!showPopup) return;

    const interval = setInterval(() => {
      setCountdown((c) => {
        if (c <= 1) {
          clearInterval(interval);
          navigate('/login');
        }
        return c - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, [showPopup, navigate]);

  const handleSelectChatroom = async (room: Chatroom) => {
    if (!user) return;

    if (selectedChatroomRef.current && socketRef.current?.readyState === WebSocket.OPEN) {
      const leavePayload: SocketPayload = {
        MsgType: 'leave',
        RoomID: selectedChatroomRef.current.ID,
        UserID: user.id,
        Message: '',
        TempID: uuidv4(),
      };
      socketRef.current.send(JSON.stringify(leavePayload));
    }

    setSelectedChatroom(room);
    selectedChatroomRef.current = room;
    setMessages([]);
    setMessagesLoading(true);
    setError(null);

    try {
      const res = await fetch(`http://${ipaddr}/chatrooms/${room.ID}/messages`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${jwttoken}`,
        },
      });

      if (!res.ok) throw new Error(`Failed to fetch messages (status ${res.status})`);

      const raw = await res.json();
      const messageList: ChatMessage[] = Array.isArray(raw) ? raw : raw.data || [];
      const modeled = messageList.map((m) => ({
        ...m,
        status: 'sent',
        fromself: m.UserID === user.id,
        TempID: uuidv4(),
      }));
      setMessages(modeled);

      if (socketRef.current?.readyState === WebSocket.OPEN) {
        const joinPayload: SocketPayload = {
          MsgType: 'join',
          RoomID: room.ID,
          UserID: user.id,
          Message: '',
          TempID: uuidv4(),
        };
        socketRef.current.send(JSON.stringify(joinPayload));
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to fetch messages');
    } finally {
      setMessagesLoading(false);
    }
  };

  const handleSendMessage = (text: string) => {
    if (!selectedChatroom || !user) return;
    if (!socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) {
      setError('Socket not connected; cannot send message.');
      return;
    }

    const payload: SocketPayload = {
      MsgType: 'message',
      RoomID: selectedChatroom.ID,
      UserID: user.id,
      Content: text,
      TempID: uuidv4(),
    };

    socketRef.current.send(JSON.stringify(payload));
    const modeledMsg: ChatMessage = {
      ID: -1,
      UserID: user.id,
      RoomID: selectedChatroom.ID,
      Content: payload.Content ?? '',
      CreatedAt: new Date().toISOString(),
      TempID: payload.TempID,
      status: 'pending',
      fromself: true,
    };
    setMessages((prev) => [...prev, modeledMsg]);
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

      {/* {error && <div className="chat-error-banner">{error}</div>}*/}
      {error && <div className="chat-error-banner">{error}</div>}
    </div>
  );
};

export default ChatRoom;
