import { useEffect, useRef } from "react";
import { createSocket } from "../services/socket";
import { UserInfo } from "../types/user";
import { KafkaEvent } from "../proto/kafka/event";

export function useChatSocket(user?: UserInfo, token?: string, onMessage?: (p: any) => void) {
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const shouldReconnectRef = useRef(true);

  useEffect(() => {
    if (!user || !token) return;

    shouldReconnectRef.current = true;

    const scheduleReconnect = () => {
      if (!shouldReconnectRef.current || reconnectTimerRef.current) return;

      reconnectAttemptsRef.current += 1;
      const baseDelayMs = 500;
      const maxDelayMs = 30_000;
      const backoff = Math.min(maxDelayMs, baseDelayMs * 2 ** (reconnectAttemptsRef.current - 1));
      const jittered = backoff * (0.5 + Math.random());

      reconnectTimerRef.current = setTimeout(() => {
        reconnectTimerRef.current = null;
        connect();
      }, jittered);
    };

    const connect = () => {
      if (!shouldReconnectRef.current) return;
      const socket = createSocket();
      socketRef.current = socket;

      socket.onopen = () => {
        reconnectAttemptsRef.current = 0;
        console.log("WebSocket connection established");
      };

      socket.onmessage = (e) => {
        try {
          const uint8Array = new Uint8Array(e.data);
          const decodedEvent = KafkaEvent.decode(uint8Array);
          onMessage?.(decodedEvent);
        } catch (err) {
          console.error("Failed to decode KafkaEvent:", err);
        }
      };

      socket.onclose = () => {
        scheduleReconnect();
      };

      socket.onerror = () => {
        scheduleReconnect();
      };
    };

    connect();
    console.log(user.username, token);
    return () => {
      shouldReconnectRef.current = false;
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }

      const current = socketRef.current;
      if (
        current &&
        (current.readyState === WebSocket.OPEN ||
          current.readyState === WebSocket.CONNECTING)
      ) {
        current.close();
      }

      socketRef.current = null;
    };
  }, [user, token, onMessage]);

  function send(payload: KafkaEvent) {
    const ws = socketRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(KafkaEvent.encode(payload).finish());
  }

  function join(roomID: number, userID: string | number) {
    const id = (BigInt(userID) * 10_000_000_000_000n + BigInt(Date.now())).toString();

    send({
      id: id,
      msgType: "join",
      roomId: roomID,
      userId: Number(userID),
      tempId: id,
      content: new Uint8Array(0),
      createdAt: String(Date.now()),
    });
  }

  function leave(roomID: number, userID: string | number) {
    const id = (BigInt(userID) * 10_000_000_000_000n + BigInt(Date.now())).toString();
    send({
      id: id,
      msgType: "leave",
      roomId: roomID,
      userId: Number(userID),
      tempId: id,
      content: new Uint8Array(0),
      createdAt: String(Date.now()),
    });
  }

  return { send, join, leave, socketRef };
}
