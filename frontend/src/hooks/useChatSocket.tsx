import { useEffect, useRef } from "react";
import { createSocket } from "../services/socket";
import { v4 as uuidv4 } from "uuid";
import { UserInfo } from "../types/user";
import { SocketPayload } from "../types/socket";
import { KafkaEvent } from "../proto/kafka/event";

export function useChatSocket(user?: UserInfo, token?: string, onMessage?: (p: any) => void) {
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!user || !token) return;

    const socket = createSocket();
    socketRef.current = socket;
    const id = (BigInt(user.id) * 10_000_000_000_000n + BigInt(Date.now())).toString();

    socket.onopen = () => {
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
    console.log(user.username, token);
    return () => {
      if (
        socket.readyState === WebSocket.OPEN ||
        socket.readyState === WebSocket.CONNECTING
      ) {
        socket.close();
      }

      socketRef.current = null;
    };
  }, [user?.id, token]);

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