import { useEffect, useRef, useState } from "react";
import { createSocket } from "../services/socket";
import { UserInfo } from "../types/user";
import { KafkaEvent } from "../proto/kafka/event";

export function useChatSocket(user?: UserInfo, token?: string, onMessage?: (p: any) => void) {
  const socketRef = useRef<WebSocket | null>(null);
  const readyRef = useRef(false);
  const roomRef = useRef<number | null>(null);
  const onMessageRef = useRef(onMessage);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => { onMessageRef.current = onMessage; }, [onMessage]);

  useEffect(() => {
    if (!user || !token) return;
    let stopped = false;
    let attempts = 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let authTimer: ReturnType<typeof setTimeout> | undefined;
    const scheduleReconnect = () => {
      if (stopped || reconnectTimer) return;
      const delay = Math.min(30_000, 500 * 2 ** attempts++) * (0.5 + Math.random());
      reconnectTimer = setTimeout(() => { reconnectTimer = undefined; connect(); }, delay);
    };
    const connect = () => {
      if (stopped) return;
      readyRef.current = false;
      const socket = createSocket();
      socketRef.current = socket;
      authTimer = setTimeout(() => socket.close(), 10_000);
      socket.onopen = () => {
        if (!stopped && socketRef.current === socket) socket.send(JSON.stringify({ type: "auth", token }));
      };
      socket.onmessage = (event) => {
        if (stopped || socketRef.current !== socket) return;
        if (typeof event.data === "string") {
          try {
            if (JSON.parse(event.data).type === "ready") {
              clearTimeout(authTimer);
              readyRef.current = true;
              attempts = 0;
              setError(null);
              if (roomRef.current !== null) socket.send(KafkaEvent.encode(roomEvent("join", roomRef.current, user.id)).finish());
            }
          } catch { setError("Invalid connection response."); socket.close(); }
          return;
        }
        if (!readyRef.current) return;
        try { onMessageRef.current?.(KafkaEvent.decode(new Uint8Array(event.data))); }
        catch { setError("Unable to read a chat message."); }
      };
      socket.onclose = (event) => {
        if (stopped || socketRef.current !== socket) return;
        clearTimeout(authTimer);
        readyRef.current = false;
        if (event.code === 1008) {
          stopped = true;
          setError("Session expired or room access denied. Sign in again or choose a room you joined.");
        } else {
          if (!stopped) setError("Connection lost. Reconnecting…");
          scheduleReconnect();
        }
      };
      socket.onerror = () => { if (stopped || socketRef.current !== socket) return; setError("Unable to connect to chat."); socket.close(); };
    };
    connect();
    return () => {
      stopped = true;
      clearTimeout(reconnectTimer);
      clearTimeout(authTimer);
      readyRef.current = false;
      socketRef.current?.close();
      socketRef.current = null;
      roomRef.current = null;
    };
  }, [user, token]);

  function send(payload: KafkaEvent) {
    const ws = socketRef.current;
    if (!readyRef.current || !ws || ws.readyState !== WebSocket.OPEN) {
      setError("Chat connection is not ready. Try again shortly.");
      return false;
    }
    ws.send(KafkaEvent.encode(payload).finish());
    return true;
  }
  function join(roomID: number, userID: string | number) {
    roomRef.current = roomID;
    if (readyRef.current) send(roomEvent("join", roomID, userID));
  }
  function leave(roomID: number, userID: string | number) {
    roomRef.current = null;
    if (readyRef.current) send(roomEvent("leave", roomID, userID));
  }
  return { send, join, leave, socketRef, error };
}

function roomEvent(msgType: string, roomId: number, userID: string | number): KafkaEvent {
  const id = (BigInt(userID) * 10_000_000_000_000n + BigInt(Date.now())).toString();
  return { id, msgType, roomId, userId: Number(userID), tempId: id, content: new Uint8Array(0), createdAt: String(Date.now()) };
}
