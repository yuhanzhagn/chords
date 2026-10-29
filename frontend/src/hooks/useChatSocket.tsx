import { useCallback, useEffect, useRef, useState } from "react";
import { createSocket } from "../services/socket";
import { UserInfo } from "../types/user";
import { KafkaEvent } from "../proto/kafka/event";

type Pending = { event: KafkaEvent; attempts: number; next: number };
// Memory only: survives reconnect/relogin/navigation in this tab, never a refresh.
// Credentials are never stored here. Switching account discards the previous queue.
let pendingOwner = "";
const pending = new Map<string, Pending>();
export function selectPendingOwner(owner: string): boolean {
  if(!owner || pendingOwner === owner) return false;
  pending.clear(); pendingOwner = owner; return true;
}

export function useChatSocket(user?: UserInfo, token?: string, onMessage?: (p: KafkaEvent) => void, onReady?: () => void) {
  const socketRef = useRef<WebSocket | null>(null);
  const readyRef = useRef(false);
  const roomRef = useRef<number | null>(null);
  const onMessageRef = useRef(onMessage);
  const onReadyRef = useRef(onReady);
  const transmitRef = useRef<(p: Pending) => boolean>(() => false);
  const owner = user ? `${user.id}/${user.username}` : "";
  const ownerRef = useRef(owner); ownerRef.current = owner;
  const [error, setError] = useState<string | null>(null);
  useEffect(() => { onMessageRef.current = onMessage; onReadyRef.current = onReady; }, [onMessage, onReady]);

  useEffect(() => {
    readyRef.current = false;
    if (selectPendingOwner(owner)) roomRef.current = null;
    if (!user || !token) return;
    let stopped = false, attempts = 0;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;
    let authTimer: ReturnType<typeof setTimeout> | undefined;
    const active = (socket: WebSocket) => !stopped && ownerRef.current === owner && socketRef.current === socket;
    const transmit = (p: Pending) => {
      const socket = socketRef.current;
      if (!socket || !active(socket) || !readyRef.current || socket.readyState !== WebSocket.OPEN || pendingOwner !== owner) return false;
      if (p.attempts >= 5) return false;
      try {
        socket.send(KafkaEvent.encode(p.event).finish());
        p.attempts++; p.next = Date.now() + Math.min(30_000, 2000 * 2 ** (p.attempts - 1));
        return true;
      } catch { socket.close(); return false; }
    };
    transmitRef.current = transmit;
    const scheduleReconnect = () => {
      if (stopped || reconnectTimer) return;
      reconnectTimer = setTimeout(() => { reconnectTimer = undefined; connect(); }, Math.min(30_000, 500 * 2 ** attempts++) * (0.5 + Math.random()));
    };
    const connect = () => {
      if (stopped) return;
      readyRef.current = false;
      const socket = createSocket(); socketRef.current = socket;
      authTimer = setTimeout(() => socket.close(), 10_000);
      socket.onopen = () => { if (active(socket)) socket.send(JSON.stringify({ type: "auth", token })); };
      socket.onmessage = event => {
        if (!active(socket)) return;
        if (typeof event.data === "string") {
          try {
            if (JSON.parse(event.data).type === "ready") {
              clearTimeout(authTimer); readyRef.current = true; attempts = 0; setError(null);
              if (roomRef.current !== null) socket.send(KafkaEvent.encode(roomEvent("join", roomRef.current, user.id)).finish());
              for (const p of pending.values()) { p.next = 0; transmit(p); }
              onReadyRef.current?.();
            }
          } catch { setError("Invalid connection response."); socket.close(); }
          return;
        }
        if (!readyRef.current) return;
        try {
          const message = KafkaEvent.decode(new Uint8Array(event.data));
          if (message.userId === user.id && message.requestId) {
            if (["persisted", "message", "deleted"].includes(message.msgType)) pending.delete(message.requestId);
            if (message.msgType === "send_error") {
              const detail = JSON.parse(new TextDecoder().decode(message.content));
              if (!detail.retryable) pending.delete(message.requestId);
              setError(detail.outcome === "unknown" ? "Send outcome unknown. Retrying with the same request ID." : `Message rejected (${detail.code}).`);
            }
          }
          onMessageRef.current?.(message);
        } catch { setError("Unable to read a chat message."); }
      };
      socket.onclose = event => {
        if (!active(socket)) return;
        clearTimeout(authTimer); readyRef.current = false;
        if (event.code === 1008) { stopped = true; setError("Session expired or room access denied. Sign in again or choose a room you joined."); }
        else if (event.code === 1009) { stopped = true; setError("Message exceeds the chat size limit. Shorten it and reconnect."); }
        else { setError("Connection lost. Reconnecting…"); scheduleReconnect(); }
      };
      socket.onerror = () => { if (active(socket)) { setError("Unable to connect to chat."); socket.close(); } };
    };
    const retryTimer = setInterval(() => {
      if (pendingOwner !== owner) return;
      for (const p of pending.values()) {
        if (p.next > Date.now()) continue;
        if (p.attempts >= 5) {
          if (p.next !== Infinity) {
            p.next = Infinity; setError("Send outcome unknown after 5 attempts. Retry pending messages or cancel waiting.");
            onMessageRef.current?.(KafkaEvent.fromPartial({ ...p.event, msgType: "send_unknown" }));
          }
        } else transmit(p);
      }
    }, 1000);
    connect();
    return () => {
      stopped = true; clearTimeout(reconnectTimer); clearTimeout(authTimer); clearInterval(retryTimer);
      readyRef.current = false; transmitRef.current = () => false;
      socketRef.current?.close(); socketRef.current = null;
    };
    // Identity primitives avoid reconnecting for an equivalent user object.
  }, [owner, user?.id, token]); // eslint-disable-line react-hooks/exhaustive-deps

  function send(payload: KafkaEvent) {
    if (!readyRef.current || !socketRef.current || socketRef.current.readyState !== WebSocket.OPEN) { setError("Chat connection is not ready. Try again shortly."); return false; }
    if (payload.msgType !== "message") { socketRef.current.send(KafkaEvent.encode(payload).finish()); return true; }
    if (payload.content.length > 256 * 1024) { setError("Message exceeds the 256 KiB UTF-8 limit."); return false; }
    if (!payload.requestId) payload.requestId = newRequestID();
    if (!pending.has(payload.requestId) && pending.size >= 128) { setError("Too many pending messages. Retry or cancel waiting first."); return false; }
    const p = pending.get(payload.requestId) || { event: KafkaEvent.fromPartial(payload), attempts: 0, next: 0 };
    pending.set(payload.requestId, p);
    return transmitRef.current(p);
  }
  function join(roomID: number, userID: string | number) { roomRef.current = roomID; if (readyRef.current) send(roomEvent("join", roomID, userID)); }
  function leave(roomID: number, userID: string | number) { if (roomRef.current === roomID) roomRef.current = null; if (readyRef.current) send(roomEvent("leave", roomID, userID)); }
  function retryPending() {
    if (pendingOwner !== owner) return;
    for (const p of pending.values()) { p.attempts = 0; p.next = 0; transmitRef.current(p); }
  }
  function cancelPending() {
    if (pendingOwner !== owner) return;
    // Cancels local retries, not a possibly committed backend transaction.
    for(const p of pending.values()) onMessageRef.current?.(KafkaEvent.fromPartial({...p.event,msgType:"send_cancelled"}));
    pending.clear(); setError("Stopped waiting. Messages may already be persisted; history will reconcile.");
  }
  const restorePending = useCallback((room: number) => {
    if(pendingOwner !== owner) return;
    for(const p of pending.values()) if(p.event.roomId === room) onMessageRef.current?.(KafkaEvent.fromPartial({...p.event,msgType:"pending_restore"}));
  },[owner]);
  return { send, join, leave, socketRef, error, retryPending, cancelPending, restorePending };
}
function roomEvent(msgType: string, roomId: number, userID: string | number): KafkaEvent {
  return KafkaEvent.fromPartial({ msgType, roomId, userId: Number(userID) });
}
export function newRequestID(): string {
  const bytes = new Uint8Array(16); crypto.getRandomValues(bytes);
  return Array.from(bytes, b => b.toString(16).padStart(2, "0")).join("");
}
