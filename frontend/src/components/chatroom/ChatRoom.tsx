import { useCallback, useMemo, useState } from "react";
import { v4 as uuidv4 } from "uuid";
import ChatSidebar from "./ChatSidebar";
import ChatWindow from "./ChatWindow";
import { useChatrooms } from "../../hooks/useChatrooms";
import { useMessages } from "../../hooks/useMessages";
import { useChatSocket } from "../../hooks/useChatSocket";
import './chat.css'
import { KafkaEvent } from "../../proto/kafka/event";

export default function ChatRoom() {
  const storedUser = localStorage.getItem("user");
  const token = localStorage.getItem("jwt") || undefined;

  const user = useMemo(() => (storedUser ? JSON.parse(storedUser) : null), [storedUser]);

  const { chatrooms, loading } = useChatrooms(user?.username, token);
  const [room, setRoom] = useState<any>(null);
  const msgStore = useMessages(room?.ID, user?.id, token);
  
  const handleSocketMessage = useCallback((payload: KafkaEvent) => {
  if (payload.msgType === "message") {
    console.log("Received message event:", payload);
    msgStore.confirm({
      ID: payload.id,
      UserID: payload.userId,
      RoomID: payload.roomId,
      Content: new TextDecoder().decode(payload.content),
      CreatedAt: new Date(payload.createdAt).toISOString(),
      TempID: payload.tempId,
      status: "sent",
      fromself: payload.userId === user.id,
    });
  }
}, [msgStore, user.id])

  const socket = useChatSocket(user, token, handleSocketMessage);

  async function selectRoom(r: any) {
    if (!user) return;
    if (room) socket.leave(room.ID, user.id);
    setRoom(r);
    msgStore.clear();
    socket.join(r.ID, user.id);
  }

  function send(text: string) {
    if (!room || !user) return;
    const id = (BigInt(user.id) * 10_000_000_000_000n + BigInt(Date.now())).toString();

    socket.send({
      id: id,
      msgType: "message",
      roomId: room.ID,
      userId: user.id,
      content: new TextEncoder().encode(text),
      tempId: id,
      createdAt: String(Date.now()),
    });

    msgStore.add({
      ID: id,
      UserID: user.id,
      RoomID: room.ID,
      Content: text,
      CreatedAt: new Date().toISOString(),
      TempID: id,
      status: "pending",
      fromself: true,
    });
  }

  return (
    <div className="chat-layout">
      <ChatSidebar
        chatrooms={chatrooms}
        loading={loading}
        selectedChatroom={room}
        onSelectChatroom={selectRoom}
      />

      <ChatWindow
        chatroom={room}
        messages={msgStore.messages}
        loading={msgStore.loading}
        onSendMessage={send}
      />
    </div>
  );
}
