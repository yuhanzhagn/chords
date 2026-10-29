import { useCallback, useEffect, useMemo, useState } from "react";
import ChatSidebar from "./ChatSidebar";
import ChatWindow from "./ChatWindow";
import { useChatrooms } from "../../hooks/useChatrooms";
import { useMessages } from "../../hooks/useMessages";
import { useChatSocket } from "../../hooks/useChatSocket";
import { Card } from "../ui/card";
import { KafkaEvent } from "../../proto/kafka/event";

function toIsoTime(createdAt: string): string {
  const raw = Number(createdAt);
  if (!Number.isFinite(raw) || raw <= 0) {
    return new Date().toISOString();
  }

  const millis = raw < 1_000_000_000_000 ? raw * 1000 : raw;
  const date = new Date(millis);
  return Number.isNaN(date.getTime()) ? new Date().toISOString() : date.toISOString();
}

export default function ChatRoom() {
  const storedUser = localStorage.getItem("user");
  const token = localStorage.getItem("jwt") || undefined;

  const user = useMemo(() => (storedUser ? JSON.parse(storedUser) : null), [storedUser]);

  const { chatrooms, loading } = useChatrooms(user?.username, token);
  const [room, setRoom] = useState<any>(null);
  const msgStore = useMessages(room?.ID, user?.id, token);

  const handleSocketMessage = useCallback(
    (payload: KafkaEvent) => {
      if (["message","persisted"].includes(payload.msgType)) {
        msgStore.confirm({
          ID: payload.serverMessageId !== "0" ? payload.serverMessageId : payload.id,
          UserID: payload.userId,
          RoomID: payload.roomId,
          Content: new TextDecoder().decode(payload.content),
          CreatedAt: payload.persistedAt || toIsoTime(payload.createdAt),
          EventID: payload.eventId,
          RoomSeq: payload.roomSeq,
          RequestID: payload.requestId,
          TempID: payload.tempId,
          status: "sent",
          fromself: payload.userId === user?.id,
        });
      }
      if(payload.msgType === "pending_restore") msgStore.add({ID:payload.tempId || payload.requestId,UserID:payload.userId,RoomID:payload.roomId,Content:new TextDecoder().decode(payload.content),CreatedAt:toIsoTime(payload.createdAt),TempID:payload.tempId,RequestID:payload.requestId,status:"pending",fromself:true});

      if (payload.msgType === "deleted") { msgStore.remove(payload.requestId); msgStore.refresh(); }
      if (["send_error", "send_unknown", "send_cancelled"].includes(payload.msgType)) {
        const unknown = payload.msgType !== "send_error" || JSON.parse(new TextDecoder().decode(payload.content)).outcome === "unknown";
        msgStore.add({ID:payload.tempId,UserID:payload.userId,RoomID:payload.roomId,Content:"",CreatedAt:"",TempID:payload.tempId,RequestID:payload.requestId,WaitingCanceled:payload.msgType==="send_cancelled",status:unknown?"unknown":"failed",fromself:payload.userId===user?.id});
      }

    },
    [msgStore, user?.id]
  );

  const socket = useChatSocket(user, token, handleSocketMessage, msgStore.refresh);

  const restorePending = socket.restorePending;
  useEffect(()=>{ if(room?.ID) restorePending(room.ID); },[room?.ID,restorePending,token]);

  async function selectRoom(r: any) {
    if (!user) return;
    if (room) socket.leave(room.ID, user.id);
    setRoom(r);
    msgStore.clear();
    socket.join(r.ID, user.id);
  }

  function send(text: string) {
    if (!room || !user) return false;
    const id = (BigInt(user.id) * 10_000_000_000_000n + BigInt(Date.now())).toString();

    const payload = KafkaEvent.fromPartial({
      id: id,
      msgType: "message",
      roomId: room.ID,
      userId: user.id,
      content: new TextEncoder().encode(text),
      tempId: id,
      createdAt: String(Date.now()),
    });

    if (!socket.send(payload)) return false;
    msgStore.add({
      ID: id,
      UserID: user.id,
      RoomID: room.ID,
      Content: text,
      CreatedAt: new Date().toISOString(),
      TempID: id,
      RequestID: payload.requestId,
      status: "pending",
      fromself: true,
    });
    return true;
  }

  return (
    <Card className="overflow-hidden border-border/70 bg-card/95">
      {(socket.error || msgStore.error) && <p role="alert" className="px-4 py-2 text-sm text-destructive">{socket.error || msgStore.error}</p>}
      {msgStore.messages.some(m => m.fromself && !m.WaitingCanceled && (m.status === "pending" || m.status === "unknown")) && <div className="px-4 py-2 text-sm"><button onClick={socket.retryPending}>Retry pending messages</button>{" · "}<button onClick={socket.cancelPending}>Cancel waiting</button><span> (canceling does not undo a persisted message)</span></div>}
      <div className="grid h-[70vh] grid-cols-1 lg:grid-cols-[280px_1fr]">
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
          loadingMore={msgStore.loadingMore}
          hasMore={msgStore.hasMore}
          onSendMessage={send}
          onLoadMore={msgStore.loadMore}
        />
      </div>
    </Card>
  );
}
